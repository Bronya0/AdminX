from django.contrib import admin
from django.urls import include, path
from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerView

api_patterns = [
    path("accounts/", include("djangoadminx.accounts.urls")),
    path("menu/", include("djangoadminx.menu.urls")),
    path("config/", include("djangoadminx.config_center.urls")),
    path("monitor/", include("djangoadminx.monitor.urls")),
    path("cluster/", include("djangoadminx.cluster.urls")),
    path("jobs/", include("djangoadminx.jobs.job_urls")),
    path("job-logs/", include("djangoadminx.jobs.job_log_urls")),
    path("common/", include("djangoadminx.common.urls")),
    path("audit/", include("djangoadminx.audit.urls")),
    path("files/", include("djangoadminx.file_center.urls")),
    path("policy/", include("djangoadminx.policy.urls")),
    path("captcha/", include("djangoadminx.captcha.urls")),
    path("data/", include("djangoadminx.data_center.urls")),
    path("notification/", include("djangoadminx.notification.urls")),
]

urlpatterns = [
    path("admin/", admin.site.urls),
    # API v1 (主要入口)
    path("api/v1/", include(api_patterns)),
    # 向后兼容: /api/xxx → /api/v1/xxx
    path("api/", include(api_patterns)),
    # API docs
    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),
    path("api/docs/", SpectacularSwaggerView.as_view(url_name="schema"), name="swagger-ui"),
]