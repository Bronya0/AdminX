from django.urls import path
from . import views

urlpatterns = [
    path("policy/", views.password_policy, name="password_policy"),
    path("change-password/", views.change_password, name="change_password"),
]