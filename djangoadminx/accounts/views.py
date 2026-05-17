import fnmatch
import json
import logging
from datetime import timedelta

from django.conf import settings
from django.contrib.auth import authenticate
from django.contrib.auth.models import Permission
from django.db.models import Q
from django.utils import timezone
from rest_framework import status, viewsets
from rest_framework.views import APIView
from rest_framework.decorators import action
from rest_framework.permissions import AllowAny, IsAdminUser, IsAuthenticated
from rest_framework.response import Response
from rest_framework_simplejwt.exceptions import TokenError
from rest_framework_simplejwt.tokens import AccessToken, RefreshToken
from rest_framework_simplejwt.views import TokenObtainPairView

from djangoadminx.audit.mixins import AuditLogMixin
from .models import BusinessCommand, BusinessPermission, LoginLock, Role, User, UserLoginLog
from djangoadminx.captcha.views import verify_captcha
from .serializers import (
    BusinessCommandSerializer,
    BusinessPermissionSerializer,
    LoginLogSerializer,
    LoginSerializer,
    PermissionSerializer,
    RoleSerializer,
    UserCreateSerializer,
    UserSerializer,
)

logger = logging.getLogger("djangoadminx.accounts")


def _record_login_log(user, request, success, message=""):
    UserLoginLog.objects.create(
        user=user if success else None,
        username=request.data.get("username", ""),
        ip=request.META.get("REMOTE_ADDR", ""),
        user_agent=request.META.get("HTTP_USER_AGENT", ""),
        success=success,
        message=message,
    )


def _check_login_lock(username):
    """检查登录锁定"""
    if not getattr(settings, "LOGIN_LOCK_ENABLED", True):
        return None
    try:
        lock = LoginLock.objects.get(username=username)
        if lock.locked_at and lock.unlocked_at is None:
            lock_duration = getattr(settings, "LOGIN_LOCK_DURATION", timedelta(minutes=15))
            if timezone.now() - lock.locked_at > lock_duration:
                lock.delete()
                return None
            remaining = int((lock_duration - (timezone.now() - lock.locked_at)).total_seconds())
            return remaining
    except LoginLock.DoesNotExist:
        pass
    return None


def _record_login_failure(username):
    from django.db.models import F
    lock, created = LoginLock.objects.get_or_create(
        username=username,
        defaults={"failed_count": 1},
    )
    if not created:
        lock.failed_count = F("failed_count") + 1
        lock.save(update_fields=["failed_count"])
        lock.refresh_from_db(fields=["failed_count"])
    max_attempts = getattr(settings, "LOGIN_MAX_ATTEMPTS", 5)
    if lock.failed_count >= max_attempts:
        LoginLock.objects.filter(username=username).update(locked_at=timezone.now())
    return lock


def _clear_login_lock(username):
    LoginLock.objects.filter(username=username).delete()


class TokenIntrospectView(APIView):
    """Token introspection — 供业务容器验证 JWT 并获取用户身份/权限

    业务容器在收到前端请求后，将 JWT 转发给此接口。
    平台返回用户身份、角色、权限，业务容器据此执行本地鉴权。
    """
    permission_classes = [AllowAny]
    authentication_classes = []  # 业务容器没有用户上下文，裸调

    # 平台登录页地址，业务容器收到 401 后可引导用户跳转
    login_url = getattr(settings, "LOGIN_URL", "/login")

    def _error(self, code: str, msg: str, http_status: int = 401, extra: dict | None = None):
        data = {"valid": False, "error_code": code, "error": msg, "login_url": self.login_url}
        if extra:
            data.update(extra)
        return Response({"code": http_status, "msg": msg, "data": data})

    def post(self, request):
        token_str = request.data.get("token", "")
        if not token_str:
            return self._error("token_missing", "未提供令牌")

        try:
            access_token = AccessToken(token_str)
        except TokenError as e:
            err_msg = str(e).lower()
            if "expired" in err_msg:
                return self._error("token_expired", "令牌已过期，请重新登录")
            if any(kw in err_msg for kw in ("wrong type", "类型错误")):
                return self._error("token_type_error", "不能使用 refresh token 调用，请传入 access token")
            return self._error("token_invalid", "令牌无效", extra={"detail": str(e)})

        user_id = access_token.payload.get("user_id")
        if not user_id:
            return self._error("token_invalid", "令牌载荷异常：缺少用户标识")

        try:
            user = User.objects.get(id=user_id)
        except User.DoesNotExist:
            return self._error("user_not_found", "用户不存在或已被删除")

        if not user.is_active:
            return self._error("user_inactive", "账号已被禁用，请联系管理员")

        if user.last_logout:
            iat = access_token.payload.get("iat")
            if iat and iat < user.last_logout.timestamp():
                return self._error(
                    "token_expired",
                    "令牌已失效，请重新登录",
                )

        # 权限：合并菜单 permission_code + BusinessPermission codename
        from djangoadminx.menu.models import Menu

        if user.is_superuser:
            perms = sorted(
                Menu.objects.filter(is_active=True)
                .exclude(permission_code="")
                .values_list("permission_code", flat=True)
            )
            biz_perms = list(
                BusinessPermission.objects.values_list("codename", flat=True)
            )
        else:
            role_ids = list(user.roles.values_list("id", flat=True))
            perms = sorted(
                Menu.objects.filter(
                    is_active=True, roles__id__in=role_ids
                ).exclude(permission_code="")
                .values_list("permission_code", flat=True)
            )
            biz_perms = list(
                BusinessPermission.objects.filter(roles__in=user.roles.all())
                .values_list("codename", flat=True)
            )

        # 角色
        roles = list(user.roles.values("name"))

        # 路径白名单检查：业务容器可校验用户是否授权访问指定 REST 路径
        check_path = request.data.get("path", "")
        check_method = request.data.get("method", "")
        path_allowed = True
        if check_path and not user.is_superuser:
            role_ids = list(user.roles.values_list("id", flat=True))
            # 只检查用户角色有权限的菜单关联的命令
            from djangoadminx.menu.models import Menu
            user_menu_paths = set(
                Menu.objects.filter(is_active=True, roles__id__in=role_ids)
                .values_list("path", flat=True)
            )
            commands = BusinessCommand.objects.filter(
                is_active=True,
                menu_path__in=user_menu_paths,
            )
            path_allowed = False
            for cmd in commands:
                try:
                    paths = json.loads(cmd.allowed_paths) if cmd.allowed_paths else []
                except json.JSONDecodeError:
                    continue
                for rule in paths:
                    if not isinstance(rule, str):
                        continue
                    rl = rule.strip()
                    rl_method = ""
                    rl_path = rl
                    if check_method and ":" in rl:
                        parts = rl.split(":", 1)
                        rl_method = parts[0].upper()
                        rl_path = parts[1].strip()
                    if rl_method and rl_method != check_method.upper():
                        continue
                    if fnmatch.fnmatch(check_path, rl_path):
                        path_allowed = True
                        break
                if path_allowed:
                    break

        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "valid": True,
                "user_id": str(user.id),
                "username": user.username,
                "email": user.email,
                "phone": user.phone,
                "avatar": user.avatar,
                "is_superuser": user.is_superuser,
                "roles": roles,
                "role_names": [r.get("name") for r in roles],
                "permissions": perms,
                "business_permissions": biz_perms,
                "exp": access_token.payload.get("exp"),
                "iat": access_token.payload.get("iat"),
                "token_type": access_token.payload.get("token_type"),
                "login_url": self.login_url,
                "path_allowed": path_allowed,
            },
        })


class LoginView(TokenObtainPairView):
    """登录 — JWT Token"""
    permission_classes = [AllowAny]
    serializer_class = LoginSerializer

    def post(self, request, *args, **kwargs):
        ser = self.get_serializer(data=request.data)
        ser.is_valid(raise_exception=True)

        username = ser.validated_data["username"]
        password = ser.validated_data["password"]

        # 验证码校验（可选）
        if getattr(settings, "CAPTCHA_ENABLED", False):
            captcha_id = ser.validated_data.get("captcha_id", "")
            captcha_text = ser.validated_data.get("captcha_text", "")
            if not verify_captcha(captcha_id, captcha_text):
                _record_login_log(None, request, False, "验证码错误")
                return Response({
                    "code": 400, "msg": "验证码错误或已过期",
                }, status=200)

        # 检查锁定
        remaining = _check_login_lock(username)
        if remaining is not None:
            _record_login_log(None, request, False, f"账号已锁定，请 {remaining}s 后重试")
            return Response({
                "code": 423, "msg": f"账号已锁定，请 {remaining}s 后重试",
            }, status=200)

        user = authenticate(username=username, password=password)
        if user is None:
            _record_login_failure(username)
            _record_login_log(None, request, False, "用户名或密码错误")
            return Response({
                "code": 401, "msg": "用户名或密码错误",
            }, status=200)

        if not user.is_active:
            _record_login_log(user, request, False, "账号已被禁用")
            return Response({
                "code": 403, "msg": "账号已被禁用",
            }, status=200)

        _clear_login_lock(username)
        refresh = RefreshToken.for_user(user)
        _record_login_log(user, request, True, "登录成功")

        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "access": str(refresh.access_token),
                "refresh": str(refresh),
                "user": UserSerializer(user).data,
            },
        })


class LogoutView(APIView):
    """登出 — 黑名单 refresh token"""
    permission_classes = [IsAuthenticated]

    def post(self, request):
        try:
            refresh_token = request.data.get("refresh")
            if refresh_token:
                token = RefreshToken(refresh_token)
                token.blacklist()
            if request.user.is_authenticated:
                from django.utils import timezone
                request.user.last_logout = timezone.now()
                request.user.save(update_fields=["last_logout"])
        except Exception as e:
            logger.warning(f"logout blacklist error: {e}")
        return Response({"code": 200, "msg": "success", "data": None})


class UserViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """用户 CRUD"""
    queryset = User.objects.prefetch_related("roles").all()
    search_fields = ["username", "email", "phone", "desc"]
    ordering_fields = ["date_joined", "username"]
    filterset_fields = ["is_active"]

    def perform_destroy(self, instance):
        from rest_framework_simplejwt.token_blacklist.models import OutstandingToken
        OutstandingToken.objects.filter(user=instance).delete()
        instance.delete()

    def filter_queryset(self, queryset):
        queryset = super().filter_queryset(queryset)
        role = self.request.query_params.get("role")
        if role:
            queryset = queryset.filter(roles__name=role)
        is_online = self.request.query_params.get("is_online")
        if is_online == "true":
            queryset = queryset.filter(last_activity__gte=timezone.now() - timedelta(minutes=5))
        elif is_online == "false":
            queryset = queryset.filter(
                Q(last_activity__lt=timezone.now() - timedelta(minutes=5)) | Q(last_activity__isnull=True)
            )
        return queryset

    def get_serializer_class(self):
        if self.action == "create":
            return UserCreateSerializer
        return UserSerializer

    @action(detail=False, methods=["get", "patch"], permission_classes=[IsAuthenticated])
    def me(self, request):
        """当前用户信息 + 权限 + 菜单"""
        user = request.user

        if request.method == "PATCH":
            ser = UserSerializer(user, data=request.data, partial=True)
            ser.is_valid(raise_exception=True)
            ser.save()
            return Response({"code": 200, "msg": "success", "data": UserSerializer(user).data})

        from djangoadminx.menu.models import Menu

        if user.is_superuser:
            menus_qs = Menu.objects.filter(is_active=True).order_by("sort_order")
        else:
            role_ids = user.roles.values_list("id", flat=True)
            menus_qs = Menu.objects.filter(
                is_active=True, roles__id__in=role_ids
            ).distinct().order_by("sort_order")

        from djangoadminx.menu.serializers import MenuFlatSerializer
        menu_ser = MenuFlatSerializer(menus_qs, many=True)

        # 权限：合并菜单 permission_code + 业务权限 codename
        perm_set: set[str] = set()
        if user.is_superuser:
            perm_set.update(
                Menu.objects.filter(is_active=True)
                .exclude(permission_code="")
                .values_list("permission_code", flat=True)
            )
            from .models import BusinessPermission
            perm_set.update(
                BusinessPermission.objects.values_list("codename", flat=True)
            )
        else:
            role_ids = list(user.roles.values_list("id", flat=True))
            perm_set.update(
                Menu.objects.filter(
                    is_active=True, roles__id__in=role_ids
                ).exclude(permission_code="")
                .values_list("permission_code", flat=True)
            )
            from .models import BusinessPermission
            perm_set.update(
                BusinessPermission.objects.filter(
                    roles__in=user.roles.all()
                ).values_list("codename", flat=True)
            )

        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "user": UserSerializer(user).data,
                "permissions": sorted(perm_set),
                "menus": menu_ser.data,
            },
        })

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def roles(self, request):
        """角色选项列表 — 用于用户管理页面的角色下拉框"""
        qs = Role.objects.filter(is_active=True).values("name")
        return Response({"code": 200, "msg": "success", "data": list(qs)})


class RoleViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """角色 CRUD"""
    queryset = Role.objects.order_by("-created_at")
    serializer_class = RoleSerializer
    search_fields = ["name", "desc"]
    ordering_fields = ["name", "created_at"]
    filterset_fields = ["is_active"]

    def filter_queryset(self, queryset):
        queryset = super().filter_queryset(queryset)
        desc = self.request.query_params.get("desc")
        if desc:
            queryset = queryset.filter(desc__icontains=desc)
        return queryset

    def _check_system_role(self, role):
        if role.is_system:
            from rest_framework.exceptions import PermissionDenied
            raise PermissionDenied("系统内置角色不可编辑或删除")

    def perform_update(self, serializer):
        role = self.get_object()
        self._check_system_role(role)
        serializer.save()

    def perform_destroy(self, instance):
        self._check_system_role(instance)
        instance.delete()

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def menu_tree(self, request):
        """菜单树 — 用于角色管理页面的菜单权限分配"""
        from djangoadminx.menu.models import Menu
        from djangoadminx.menu.serializers import MenuTreeSerializer
        if hasattr(Menu, 'get_root_nodes'):
            menus = Menu.get_root_nodes().filter(is_active=True, is_visible=True)
        else:
            menus = Menu.objects.filter(parent__isnull=True, is_active=True, is_visible=True)
        ser = MenuTreeSerializer(menus, many=True)
        return Response({"code": 200, "msg": "success", "data": ser.data})

    @action(detail=False, methods=["post"], permission_classes=[IsAuthenticated])
    def accessible_menus(self, request):
        """根据角色名列表返回可访问的菜单路径（用于用户首页配置）"""
        role_names = request.data.get("roles", [])
        from djangoadminx.menu.models import Menu
        from djangoadminx.menu.serializers import MenuFlatSerializer
        menus = Menu.objects.filter(is_active=True, is_visible=True, numchild=0).exclude(path="")
        if role_names:
            menus = menus.filter(roles__name__in=role_names)
        menus = menus.distinct().order_by("sort_order")
        ser = MenuFlatSerializer(menus, many=True)
        return Response({"code": 200, "msg": "success", "data": ser.data})


class BusinessPermissionViewSet(viewsets.ModelViewSet):
    """业务权限 — 供业务容器注册/查询"""
    queryset = BusinessPermission.objects.all()
    serializer_class = BusinessPermissionSerializer
    search_fields = ["name", "codename", "app_label"]
    ordering_fields = ["app_label", "codename"]
    filterset_fields = ["app_label"]
    pagination_class = None  # 列表不分页


class PermissionViewSet(viewsets.ReadOnlyModelViewSet):
    """权限列表"""
    queryset = Permission.objects.select_related("content_type").all()
    serializer_class = PermissionSerializer
    search_fields = ["name", "codename"]
    ordering_fields = ["content_type__name"]
    pagination_class = None  # 权限是有限数据集，不分页


class LoginLogViewSet(viewsets.ReadOnlyModelViewSet):
    """登录日志"""
    queryset = UserLoginLog.objects.all()
    serializer_class = LoginLogSerializer
    search_fields = ["username", "ip", "message"]
    ordering_fields = ["created_at"]
    ordering = ["-created_at"]
    filterset_fields = ["success"]

    def filter_queryset(self, queryset):
        queryset = super().filter_queryset(queryset)
        date_from = self.request.query_params.get("created_at__gte")
        if date_from:
            queryset = queryset.filter(created_at__gte=date_from)
        date_to = self.request.query_params.get("created_at__lte")
        if date_to:
            queryset = queryset.filter(created_at__lte=date_to)
        ip = self.request.query_params.get("ip")
        if ip:
            queryset = queryset.filter(ip__icontains=ip)
        return queryset


class BusinessCommandViewSet(viewsets.ModelViewSet):
    """业务命令 — 供业务容器注册菜单 + 路径白名单"""
    queryset = BusinessCommand.objects.all()
    serializer_class = BusinessCommandSerializer
    search_fields = ["name", "app_label", "menu_path"]
    ordering_fields = ["app_label", "name"]
    filterset_fields = ["app_label", "is_active"]
    pagination_class = None  # 数据量小，不分页