import io
import logging
import random
import string

from django.conf import settings
from django.core.cache import cache
from django.http import HttpResponse
from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import AllowAny
from rest_framework.response import Response

logger = logging.getLogger("djangoadminx.captcha")

# 简单验证码 — 不依赖 PIL，纯字符方式
# 真正生产环境建议装 Pillow + 图形验证码


def generate_captcha_text(length=4):
    """生成随机验证码文本"""
    chars = string.digits + string.ascii_uppercase
    # 排除易混淆字符
    chars = chars.replace("0", "").replace("O", "").replace("I", "").replace("l", "")
    return "".join(random.choices(chars, k=length))


@api_view(["GET"])
@permission_classes([AllowAny])
def captcha_image(request):
    """获取验证码 — 返回 SVG 格式纯文本

    生产环境建议:
      1. pip install Pillow
      2. 用 ImageDraw 生成带噪点的图形验证码
    """
    text = generate_captcha_text()

    # 存到 cache（带前缀，有效期 5 分钟）
    captcha_id = "".join(random.choices(string.ascii_lowercase + string.digits, k=16))
    cache.set(f"captcha:{captcha_id}", text, timeout=300)

    # 生成简单 SVG
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" width="120" height="40">
  <rect width="120" height="40" fill="#f0f0f0" rx="4"/>
  <text x="60" y="28" text-anchor="middle"
        font-family="monospace" font-size="22"
        fill="#333" letter-spacing="4">{text}</text>
</svg>'''

    response = HttpResponse(svg, content_type="image/svg+xml")
    response["X-Captcha-Id"] = captcha_id
    return response


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
    """验证验证码 — 前端在登录前先调此接口校验"""
    captcha_id = request.data.get("captcha_id", "")
    captcha_text = request.data.get("captcha_text", "")
    ok = verify_captcha(captcha_id, captcha_text)
    return Response({
        "code": 200 if ok else 400,
        "msg": "验证码正确" if ok else "验证码错误或已过期",
        "data": {"verified": ok},
    })