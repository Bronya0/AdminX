from rest_framework.renderers import JSONRenderer


class StandardJsonRenderer(JSONRenderer):
    """统一 JSON 响应格式: {code, msg, data}

    规则:
      - 正常响应 (2xx): 自动包装为 {code, msg, data}
      - 错误响应 (4xx/5xx): 自动包装为 {code, msg, data}
      - 如果 data 已经是 {code, ..., msg, ...} 格式, 不再重复包装
    """

    def render(self, data, accepted_media_type=None, renderer_context=None):
        response = renderer_context.get("response") if renderer_context else None
        status_code = response.status_code if response else 200

        # 已经是标准格式 → 直接返回
        if isinstance(data, dict) and "code" in data and "msg" in data:
            return super().render(data, accepted_media_type, renderer_context)

        # 已经是 {data: ..., ...} 包含常用包装字段 → 直接返回
        # (兼容 pagination 返回的 {list, total, page, size} 等)
        # 不包装, 因为用户自定义格式

        if status_code >= 400 and isinstance(data, dict):
            detail = data.pop("detail", None) or next(
                (str(v[0]) if isinstance(v, list) else str(v))
                for v in data.values()
            )
            wrapped = {"code": status_code, "msg": detail, "data": None}
        else:
            wrapped = {"code": status_code, "msg": "success", "data": data}

        renderer_context["response"].status_code = 200
        return super().render(wrapped, accepted_media_type, renderer_context)