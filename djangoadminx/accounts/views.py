import logging
from datetime import timedelta

from django.conf import settings
from django.contrib.auth import authenticate
from django.contrib.auth.models import Permission
from django.utils import timezone
from rest_framework import status, viewsets
from rest_framework.views import APIView
from rest_framework.decorators import action
from rest_framework.permissions import AllowAny, IsAdminUser, IsAuthenticated
from rest_framework.response import Response
from rest_framework_simplejwt.tokens import RefreshToken
from rest_framework_simplejwt.views import TokenObtainPairView

from .models import LoginLock, Role, User, UserLoginLog
from ..captcha.views import verify_captcha
from .serializers import (
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
    lock, _ = LoginLock.objects.get_or_create(username=username)
    lock.failed_count += 1
    max_attempts = getattr(settings, "LOGIN_MAX_ATTEMPTS", 5)
    if lock.failed_count >= max_attempts:
        lock.locked_at = timezone.now()
    lock.save()


def _clear_login_lock(username):
    LoginLock.objects.filter(username=username).delete()


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
        except Exception as e:
            logger.warning(f"logout blacklist error: {e}")
        return Response({"code": 200, "msg": "success"})


class UserViewSet(viewsets.ModelViewSet):
    """用户 CRUD"""
    queryset = User.objects.all()
    search_fields = ["username", "email", "phone"]
    ordering_fields = ["date_joined", "username"]

    def get_serializer_class(self):
        if self.action == "create":
            return UserCreateSerializer
        return UserSerializer

    @action(detail=False, methods=["get"], permission_classes=[IsAuthenticated])
    def me(self, request):
        """当前用户信息 + 权限 + 菜单"""
        user = request.user
        perms = user.get_all_permissions()
        if user.is_superuser:
            from djangoadminx.menu.models import Menu
            menus_qs = Menu.objects.filter(is_active=True).order_by("sort_order")
        else:
            from djangoadminx.menu.models import Menu
            role_ids = user.roles.values_list("id", flat=True)
            menus_qs = Menu.objects.filter(
                is_active=True, roles__id__in=role_ids
            ).distinct().order_by("sort_order")

        from djangoadminx.menu.serializers import MenuFlatSerializer
        menu_ser = MenuFlatSerializer(menus_qs, many=True)

        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "user": UserSerializer(user).data,
                "permissions": list(perms),
                "menus": menu_ser.data,
            },
        })


class RoleViewSet(viewsets.ModelViewSet):
    """角色 CRUD"""
    queryset = Role.objects.all()
    serializer_class = RoleSerializer
    search_fields = ["name", "code"]
    ordering_fields = ["name", "created_at"]


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
    ordering_fields = ["created_at"]
    ordering = ["-created_at"]