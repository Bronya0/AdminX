from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views

router = DefaultRouter()
router.register("jobs", views.ScheduleJobViewSet)
router.register("job-logs", views.JobLogViewSet)

urlpatterns = [
    path("", include(router.urls)),
]
