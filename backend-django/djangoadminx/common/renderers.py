from rest_framework.renderers import JSONRenderer


class StandardJsonRenderer(JSONRenderer):
    """统一 JSON 响应格式: {code, msg, data}"""

    def render(self, data, accepted_media_type=None, renderer_context=None):
        response = renderer_context.get("response") if renderer_context else None
        status_code = response.status_code if response else 200

        if isinstance(data, dict) and "code" in data and "msg" in data:
            return super().render(data, accepted_media_type, renderer_context)

        if status_code >= 400 and isinstance(data, dict):
            detail = data.pop("detail", None)
            if not detail:
                vals = [v for v in data.values() if v is not None]
                if vals:
                    first = vals[0]
                    detail = str(first[0]) if isinstance(first, list) else str(first)
                else:
                    detail = "请求错误"
            wrapped = {"code": status_code, "msg": detail, "data": None}
        else:
            wrapped = {"code": status_code, "msg": "success", "data": data}

        renderer_context["response"].status_code = 200
        return super().render(wrapped, accepted_media_type, renderer_context)
