"""WSDL 发布 — 使用 spyne 对外暴露 Django Model 数据的 SOAP 接口"""

import logging

from django.conf import settings
from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from spyne import Application, srpc, ServiceBase
from spyne.model.complex import Iterable
from spyne.model.primitive import String
from spyne.protocol.soap import Soap11
from spyne.server.django import DjangoView

logger = logging.getLogger("djangoadminx.webservice.publish")


class UserDataService(ServiceBase):
    """用户数据查询服务"""

    @srpc(String, _returns=Iterable(String))
    def list_users(keyword=""):
        """获取用户列表，支持按用户名/邮箱模糊搜索"""
        from djangoadminx.accounts.models import User

        qs = User.objects.all().only("id", "username", "email", "phone", "is_active", "date_joined")
        if keyword:
            qs = qs.filter(username__icontains=keyword) | qs.filter(email__icontains=keyword)

        for u in qs.order_by("-date_joined")[:100]:
            yield (
                f"id={u.id}|username={u.username}|email={u.email or ''}|"
                f"phone={u.phone or ''}|is_active={u.is_active}|"
                f"date_joined={u.date_joined.isoformat() if u.date_joined else ''}"
            )

    @srpc(String, _returns=String)
    def get_user(user_id=""):
        """根据 ID 获取单个用户详情"""
        from djangoadminx.accounts.models import User

        try:
            u = User.objects.get(id=user_id)
            return (
                f"id={u.id}|username={u.username}|email={u.email or ''}|"
                f"phone={u.phone or ''}|is_active={u.is_active}|"
                f"date_joined={u.date_joined.isoformat() if u.date_joined else ''}"
            )
        except User.DoesNotExist:
            return ""


class MenuDataService(ServiceBase):
    """菜单数据查询服务"""

    @srpc(String, _returns=Iterable(String))
    def list_menus(keyword=""):
        """获取菜单列表"""
        from djangoadminx.menu.models import Menu

        qs = Menu.objects.all()
        if keyword:
            qs = qs.filter(title__icontains=keyword)

        for m in qs.order_by("path", "sort_order")[:200]:
            yield (
                f"id={m.id}|name={m.name}|path={m.path or ''}|"
                f"permission_code={m.permission_code or ''}|sort_order={m.sort_order}"
            )


class ConfigDataService(ServiceBase):
    """配置数据查询服务"""

    @srpc(String, _returns=Iterable(String))
    def list_configs(keyword=""):
        """获取配置列表（不返回加密值）"""
        from djangoadminx.config_center.models import Config

        qs = Config.objects.all()
        if keyword:
            qs = qs.filter(key__icontains=keyword) | qs.filter(desc__icontains=keyword)

        for c in qs.order_by("key")[:200]:
            value_display = "***encrypted***" if c.value_type == "encrypted" else (c.value or "")
            yield f"id={c.id}|key={c.key}|value_type={c.value_type}|value={value_display}"


class ClusterDataService(ServiceBase):
    """集群节点数据查询服务"""

    @srpc(String, _returns=Iterable(String))
    def list_nodes(keyword=""):
        """获取集群节点列表"""
        from djangoadminx.cluster.models import ClusterNode

        qs = ClusterNode.objects.all()
        if keyword:
            qs = qs.filter(name__icontains=keyword) | qs.filter(host__icontains=keyword)

        for n in qs.order_by("name")[:100]:
            yield (
                f"id={n.id}|name={n.name}|host={n.host}|port={n.port}|"
                f"role={n.role or ''}|is_active={n.is_active}|"
                f"last_heartbeat={n.last_heartbeat.isoformat() if n.last_heartbeat else ''}"
            )


# ---------- 构建 spyne Application ----------

# 各个服务独立 Application，方便分别访问 WSDL
_applications = {
    "user": Application(
        [UserDataService],
        tns="http://djangoadminx.com/soap/user",
        in_protocol=Soap11(validator="lxml"),
        out_protocol=Soap11(),
        name="UserService",
    ),
    "menu": Application(
        [MenuDataService],
        tns="http://djangoadminx.com/soap/menu",
        in_protocol=Soap11(validator="lxml"),
        out_protocol=Soap11(),
        name="MenuService",
    ),
    "config": Application(
        [ConfigDataService],
        tns="http://djangoadminx.com/soap/config",
        in_protocol=Soap11(validator="lxml"),
        out_protocol=Soap11(),
        name="ConfigService",
    ),
    "cluster": Application(
        [ClusterDataService],
        tns="http://djangoadminx.com/soap/cluster",
        in_protocol=Soap11(validator="lxml"),
        out_protocol=Soap11(),
        name="ClusterService",
    ),
}


def _auth_check(request):
    """验证 JWT token，返回 (user, None) 或 (None, error_response)"""
    from rest_framework_simplejwt.authentication import JWTAuthentication

    try:
        auth = JWTAuthentication()
        user, _ = auth.authenticate(request)
        if user is None:
            return None, JsonResponse({"code": 401, "msg": "未授权访问，请提供有效的 JWT token"})
        return user, None
    except Exception:
        return None, JsonResponse({"code": 401, "msg": "未授权访问，请提供有效的 JWT token"})


def _make_soap_view(app_key):
    """工厂函数 — 创建绑定到特定 spyne Application 的 Django 视图函数"""
    app = _applications.get(app_key)
    if app is None:
        raise ValueError(f"Unknown application key: {app_key}")

    from spyne.server.django import DjangoServer
    server = DjangoServer(app, cache_wsdl=True)

    django_view = DjangoView.as_view(application=app)

    @csrf_exempt
    def view(request, **kwargs):
        # JWT 认证
        user, err = _auth_check(request)
        if err:
            return err

        # 修复 location URL — 去掉 ?wsdl 后缀，使其指向干净的 endpoint 地址
        original_url = getattr(request, 'url', None)
        if request.META.get('QUERY_STRING', '').startswith('wsdl'):
            # 对于 WSDL 请求，调整 transport_url 使 location 不包含 ?wsdl
            from urllib.parse import urlparse, urlunparse
            parsed = urlparse(request.build_absolute_uri())
            clean_url = urlunparse((parsed.scheme, parsed.netloc, parsed.path, None, None, None))
            # 通过 META 传递给 spyne
            request.META['transport_url'] = clean_url

        return django_view(request, **kwargs)

    return view


# 导出视图函数供 urls.py 引用
soap_user_view = _make_soap_view("user")
soap_menu_view = _make_soap_view("menu")
soap_config_view = _make_soap_view("config")
soap_cluster_view = _make_soap_view("cluster")