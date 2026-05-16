import io
import logging
import secrets
import string

from django.conf import settings
from django.core.cache import cache
from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import AllowAny
from rest_framework.response import Response

logger = logging.getLogger("djangoadminx.captcha")


def generate_captcha_text(length=4):
    """生成随机验证码文本"""
    chars = string.digits + string.ascii_uppercase
    chars = chars.replace("0", "").replace("O", "").replace("I", "").replace("l", "")
    return "".join(secrets.choice(chars) for _ in range(length))


@api_view(["GET"])
@permission_classes([AllowAny])
def captcha_image(request):
    """获取验证码 — 返回 JSON，内含 SVG 和 captcha_id"""
    text = generate_captcha_text()

    captcha_id = secrets.token_hex(16)
    cache.set(f"captcha:{captcha_id}", text, timeout=300)

    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="120" height="40">
  <rect width="120" height="40" fill="#f0f0f0" rx="4"/>
  <text x="60" y="28" text-anchor="middle"
        font-family="monospace" font-size="22"
        fill="#333" letter-spacing="4">{text}</text>
</svg>'''

    return Response({
        "code": 200,
        "msg": "success",
        "data": {
            "captcha_id": captcha_id,
            "svg": svg,
        },
    })


def verify_captcha(captcha_id, captcha_text):
    """校验验证码 — 验证后删除"""
    if not captcha_id or not captcha_text:
        return False
    key = f"captcha:{captcha_id}"
    expected = cache.get(key)
    if expected and expected.upper() == captcha_text.upper():
        cache.delete(key)
        return True
    return False


@api_view(["POST"])
@permission_classes([AllowAny])
def captcha_verify(request):
    """验证验证码"""
    captcha_id = request.data.get("captcha_id", "")
    captcha_text = request.data.get("captcha_text", "")
    ok = verify_captcha(captcha_id, captcha_text)
    return Response({
        "code": 200 if ok else 400,
        "msg": "验证码正确" if ok else "验证码错误或已过期",
        "data": {"verified": ok},
    })
