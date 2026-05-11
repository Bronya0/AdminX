from rest_framework.views import exception_handler


def custom_exception_handler(exc, context):
    """全局异常处理器 — 统一返回 {code, msg, data} 格式"""
    response = exception_handler(exc, context)
    if response is not None:
        detail = response.data.pop("detail", None)
        if detail:
            msg = detail
        else:
            first = next(iter(response.data.values()), None)
            msg = str(first[0]) if isinstance(first, list) else str(first)
        response.data = {"code": response.status_code, "msg": msg, "data": response.data}
        response.status_code = 200
    return response