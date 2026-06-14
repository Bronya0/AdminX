from django.urls import path
from . import views

urlpatterns = [
    path("captcha/", views.captcha_image, name="captcha"),
    path("captcha/verify/", views.captcha_verify, name="captcha_verify"),
]