import os

# 在 spyne 加载前应用兼容垫片（Python 3.13+）
import djangoadminx.common.spyne_compat  # noqa: F401

from django.core.asgi import get_asgi_application

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings.prod")

django_asgi_app = get_asgi_application()

from channels.routing import ProtocolTypeRouter, URLRouter  # noqa: E402
from djangoadminx.common import routing as common_routing  # noqa: E402

application = ProtocolTypeRouter({
    "http": django_asgi_app,
    "websocket": URLRouter(common_routing.websocket_urlpatterns),
})