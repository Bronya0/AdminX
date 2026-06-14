from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views

router = DefaultRouter()
router.register("", views.ScheduleJobViewSet)

log_router = DefaultRouter()
log_router.register("", views.JobLogViewSet)

urlpatterns = [
    path("logs/", include(log_router.urls)),
    path("", include(router.urls)),
]
