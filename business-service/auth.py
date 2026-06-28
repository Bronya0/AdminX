"""JWT introspection — 将 JWT 转发给 DjangoAdminX 平台校验"""

import logging

import httpx

from config import PLATFORM_URL

logger = logging.getLogger("business.auth")


async def introspect_token(token: str) -> dict | None:
    """将 JWT 转发给平台 introspect 接口校验，返回用户信息或 None"""
    if not token:
        return None
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(
                f"{PLATFORM_URL}/api/v1/accounts/introspect/",
                json={"token": token},
                headers={"Authorization": f"Bearer {token}"},
            )
            if resp.status_code != 200:
                logger.warning("introspect 返回非 200: %d", resp.status_code)
                return None
            body = resp.json()
    except (httpx.HTTPError, ValueError) as e:
        logger.warning("introspect 请求失败: %s", e)
        return None

    data = body.get("data", {})
    if not data.get("valid"):
        return None

    return {
        "user_id": data.get("user_id"),
        "username": data.get("username"),
        "roles": data.get("roles", []),
        "permissions": data.get("permissions", []),
    }
