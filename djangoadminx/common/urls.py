from django.urls import path
from . import views
from . import cache_views

urlpatterns = [
    path("health/", views.health_check, name="health"),
    path("site-info/", views.site_info, name="site_info"),
    path("log/tail/", views.log_tail, name="log_tail"),
    path("cache/stats/", cache_views.cache_stats, name="cache_stats"),
    path("cache/clear/", cache_views.cache_clear, name="cache_clear"),
    path("ntp/sync/", views.ntp_sync, name="ntp_sync"),
]