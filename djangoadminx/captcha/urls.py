from django.urls import path
from . import views

urlpatterns = [
    path("captcha/", views.captcha_image, name="captcha"),
]