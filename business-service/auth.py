"""JWT introspection — 将 JWT 转发给 DjangoAdminX 平台校验"""

import httpx

from config import PLATFORM_URL


async def introspect_token(token: str) -> dict | None:
    """将 JWT 转发给平台 introspect 接口校验，返回用户信息或 None"""
    if not token:
        return None
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.post(
            f"{PLATFORM_URL}/api/v1/accounts/introspect/",
            json={"token": token},
        )
        body = resp.json()
        data = body.get("data", {})

    if not data.get("valid"):
        return None

    return {
        "user_id": data.get("user_id"),
        "username": data.get("username"),
        "roles": data.get("roles", []),
        "permissions": data.get("permissions", []),
    }
