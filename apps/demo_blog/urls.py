"""
URL 配置 — 规范示例

最佳实践:
  1. 用 DefaultRouter 自动生成 RESTful 路由
  2. 每个 ViewSet 注册为 `{resource}/`
  3. 自定义 endpoint 通过 ViewSet 的 @action 实现
"""
from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views
from .publish import soap_post_view

router = DefaultRouter()
router.register("categories", views.CategoryViewSet)
router.register("posts", views.PostViewSet)

urlpatterns = [
    path("", include(router.urls)),
    # WSDL 发布 — 对外暴露 SOAP 接口
    path("publish/post/", soap_post_view, name="wsdl_post"),
]