from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views

router = DefaultRouter()
router.register("users", views.UserViewSet)
router.register("roles", views.RoleViewSet)
router.register("permissions", views.PermissionViewSet)
router.register("login-logs", views.LoginLogViewSet)

urlpatterns = [
    path("login/", views.LoginView.as_view(), name="login"),
    path("logout/", views.LogoutView.as_view(), name="logout"),
    path("", include(router.urls)),
]

# CAS SSO 路由（仅当 django-cas-ng 已安装时注册）
try:
    import django_cas_ng  # noqa: F401
    from . import sso
    urlpatterns += [
        path("cas/login/", sso.CASLoginView.as_view(), name="cas_login"),
        path("cas/logout/", sso.CASLogoutView.as_view(), name="cas_logout"),
        path("cas/callback/", sso.CASCallbackView.as_view(), name="cas_callback"),
    ]
except ImportError:
    pass