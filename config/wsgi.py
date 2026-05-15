import os

# 在 spyne 加载前应用兼容垫片（Python 3.13+）
import djangoadminx.common.spyne_compat  # noqa: F401

from django.core.wsgi import get_wsgi_application

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings.prod")
application = get_wsgi_application()