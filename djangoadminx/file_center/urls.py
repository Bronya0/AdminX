from django.urls import include, path
from rest_framework.routers import DefaultRouter

from . import views

router = DefaultRouter()
router.register("records", views.FileViewSet)

urlpatterns = [
    path("upload/", views.FileUploadView.as_view(), name="file_upload"),
    path("", include(router.urls)),
]