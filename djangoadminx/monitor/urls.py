from django.urls import path
from . import views

urlpatterns = [
    path("resources/", views.system_resource, name="monitor_resources"),
    path("netstat/", views.netstat_info, name="monitor_netstat"),
]