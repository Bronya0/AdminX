"""IP 工具函数

Django 没有内置的客户端 IP 提取方法。
生产环境通常在反向代理（Nginx / 负载均衡）后面运行，
真实 IP 由代理追加到 X-Forwarded-For 头部。

X-Forwarded-For 格式：client, proxy1, proxy2
取第一个即为真实客户端 IP。

注意：只有在可信代理环境下，XFF 头才可信。
若对外直接暴露 Django，攻击者可伪造 XFF，此时应去掉 XFF 读取逻辑。
可通过 settings.TRUST_PROXY_HEADERS = True 显式开启（默认 False）。
"""

from django.conf import settings


def get_client_ip(request) -> str:
    """从请求中提取客户端真实 IP。

    读取顺序：
    1. HTTP_X_FORWARDED_FOR（仅 TRUST_PROXY_HEADERS=True 时启用）
    2. REMOTE_ADDR（直连 IP，始终可用）
    """
    if getattr(settings, "TRUST_PROXY_HEADERS", False):
        xff = request.META.get("HTTP_X_FORWARDED_FOR", "")
        if xff:
            # XFF 可能包含多个 IP，取最左侧（真实客户端）
            return xff.split(",")[0].strip()

    return request.META.get("REMOTE_ADDR", "")
