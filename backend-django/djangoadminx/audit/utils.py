"""审计通用工具 — 被 mixins.py 和 signals.py 共享"""

import json

from djangoadminx.common.ip_utils import get_client_ip


def serialize_for_json(obj):
    """将模型实例转为 JSON 可序列化的 dict"""
    import uuid as _uuid
    result = {}
    for field in obj._meta.fields:
        val = getattr(obj, field.name)
        if hasattr(val, "isoformat"):
            val = val.isoformat() if val else None
        elif hasattr(val, "pk"):
            val = str(val.pk) if val and val.pk else None
        elif isinstance(val, _uuid.UUID):
            val = str(val)
        result[field.name] = val
    return result


def get_operator(request):
    if request and hasattr(request, "user") and request.user.is_authenticated:
        return request.user.username or str(request.user)
    return "system"


def get_operator_ip(request):
    if request:
        return get_client_ip(request)
    return ""
