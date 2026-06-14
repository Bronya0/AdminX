"""
CAS 单点登录支持 (django-cas-ng)

通过 settings.py 中的 CAS_SERVER_URL 控制开关:
  - 配置了 CAS_SERVER_URL → SSO 模式启用
  - 未配置 → 仅 JWT 密码登录
"""
import logging

from django.conf import settings
from django.contrib.auth import login as auth_login, logout as auth_logout
from django.http import JsonResponse
from rest_framework.permissions import AllowAny
from rest_framework.views import APIView

logger = logging.getLogger("djangoadminx.accounts.sso")


class CASLoginView(APIView):
    """CAS 登录入口 — 重定向到 CAS Server"""
    permission_classes = [AllowAny]

    def get(self, request):
        cas_server = getattr(settings, "CAS_SERVER_URL", None)
        if not cas_server:
            return JsonResponse({"code": 400, "msg": "CAS 未配置"})
        from django_cas_ng.views import LoginView as CASLoginView
        return CASLoginView.as_view()(request)


class CASLogoutView(APIView):
    """CAS 登出"""
    permission_classes = [AllowAny]

    def post(self, request):
        from django_cas_ng.views import LogoutView as CASLogoutView
        return CASLogoutView.as_view()(request)


class CASCallbackView(APIView):
    """CAS 回调 — 认证成功后返回 JWT Token"""
    permission_classes = [AllowAny]

    def get(self, request):
        from django_cas_ng.views import LoginView as CASLoginView
        response = CASLoginView.as_view()(request)
        # 如果 CAS 认证成功，生成 JWT
        if request.user.is_authenticated:
            from rest_framework_simplejwt.tokens import RefreshToken
            from djangoadminx.accounts.serializers import UserSerializer
            refresh = RefreshToken.for_user(request.user)
            return JsonResponse({
                "code": 200,
                "msg": "success",
                "data": {
                    "access": str(refresh.access_token),
                    "refresh": str(refresh),
                    "user": UserSerializer(request.user).data,
                },
            })
        return response