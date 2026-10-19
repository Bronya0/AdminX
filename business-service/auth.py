"""JWT introspection — 将 JWT 转发给 AdminX 平台校验（带本地 TTL 缓存）"""

import logging
import time

import httpx

from config import PLATFORM_API_BASE, INTROSPECT_CACHE_TTL

logger = logging.getLogger("business.auth")

# 本地缓存: token -> (expire_ts, user_info)。
# 缓存可避免每个业务请求都打平台 introspect（限流 300/min/IP），
# 平台短暂不可用时也能兜底放行已缓存的有效 token。
# 注意：缓存会延迟登出/封禁的生效时间（默认关闭，可用 INTROSPECT_CACHE_TTL 调整，
# 设为 0 关闭缓存）。
_cache: dict[str, tuple[float, dict]] = {}


def _cache_get(token: str) -> dict | None:
    if INTROSPECT_CACHE_TTL <= 0:
        return None
    item = _cache.get(token)
    if not item:
        return None
    expire_ts, info = item
    if time.time() >= expire_ts:
        _cache.pop(token, None)
        return None
    return info


def _cache_set(token: str, info: dict) -> None:
    if INTROSPECT_CACHE_TTL <= 0:
        return
    # 简单防膨胀：超过 1000 条时清空重建
    if len(_cache) > 1000:
        _cache.clear()
    expiry = min(time.time() + INTROSPECT_CACHE_TTL, info["exp"])
    if expiry > time.time():
        _cache[token] = (expiry, info)


async def introspect_token(token: str) -> dict | None:
    """将 JWT 转发给平台 introspect 接口校验，返回用户信息或 None"""
    if not token:
        return None

    # 命中本地缓存直接返回
    cached = _cache_get(token)
    if cached is not None:
        return cached

    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(
                f"{PLATFORM_API_BASE}/accounts/introspect/",
                json={"token": token},
                headers={"Authorization": f"Bearer {token}"},
            )
            if resp.status_code != 200:
                raise RuntimeError("平台鉴权服务返回异常状态")
            body = resp.json()
    except (httpx.HTTPError, ValueError) as e:
        logger.warning("introspect 请求失败: %s", e)
        raise RuntimeError("平台鉴权服务不可达") from e

    if not isinstance(body, dict) or not isinstance(body.get("data"), dict):
        raise RuntimeError("平台鉴权响应格式错误")
    data = body["data"]
    if data.get("valid") is not True:
        return None
    if not isinstance(data.get("user_id"), str) or not data["user_id"]:
        raise RuntimeError("平台鉴权响应缺少用户标识")

    info = {
        "user_id": data.get("user_id"),
        "username": data.get("username"),
        "roles": data.get("roles", []),
        "permissions": data.get("permissions", []),
        "exp": data.get("exp", 0),
    }
    _cache_set(token, info)
    return info
