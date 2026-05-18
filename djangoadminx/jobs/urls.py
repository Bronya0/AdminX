from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views

router = DefaultRouter()
router.register("jobs/logs", views.JobLogViewSet)
router.register("jobs", views.ScheduleJobViewSet)

urlpatterns = [
    path("", include(router.urls)),
]
