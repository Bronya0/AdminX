from rest_framework.views import exception_handler


def custom_exception_handler(exc, context):
    """全局异常处理器 — 统一返回 {code, msg, data} 格式"""
    response = exception_handler(exc, context)
    if response is not None:
        data = dict(response.data) if isinstance(response.data, dict) else {}
        detail = data.pop("detail", None)
        if detail:
            msg = detail
        else:
            vals = [v for v in data.values() if v is not None]
            if vals:
                first = vals[0]
                msg = str(first[0]) if isinstance(first, list) else str(first)
            else:
                msg = "请求错误"
        response.data = {"code": response.status_code, "msg": msg, "data": data}
        response.status_code = 200
    return response
