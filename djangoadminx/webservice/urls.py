from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views
from .publish import soap_cluster_view, soap_config_view, soap_menu_view, soap_user_view

router = DefaultRouter()
router.register("configs", views.WebServiceViewSet)
router.register("jobs", views.ScheduleJobViewSet)
router.register("job-logs", views.JobLogViewSet)
router.register("ws-logs", views.WebServiceLogViewSet)

# WSDL 发布路由 — 每个服务独立 endpoint，?wsdl 获取 WSDL
publish_paths = [
    path("publish/user/", soap_user_view, name="wsdl_user"),
    path("publish/menu/", soap_menu_view, name="wsdl_menu"),
    path("publish/config/", soap_config_view, name="wsdl_config"),
    path("publish/cluster/", soap_cluster_view, name="wsdl_cluster"),
]

urlpatterns = [
    path("", include(router.urls)),
] + publish_paths