from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views

router = DefaultRouter()
router.register("", views.NotificationViewSet, basename="notification")
router.register("webhooks", views.WebhookConfigViewSet, basename="webhook-config")
router.register("webhook-logs", views.WebhookLogViewSet, basename="webhook-log")

urlpatterns = [
    path("", include(router.urls)),
]
