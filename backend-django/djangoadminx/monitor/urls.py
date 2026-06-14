from django.urls import path
from . import views

urlpatterns = [
    path("resources/", views.system_resource, name="monitor_resources"),
    path("resources/history/", views.system_resource_history, name="monitor_resources_history"),
    path("netstat/", views.netstat_info, name="monitor_netstat"),
]
