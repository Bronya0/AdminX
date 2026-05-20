"""菜单注册 — 向 DjangoAdminX 平台注册自身菜单（幂等）"""

import logging

from fastapi import APIRouter, HTTPException
import httpx

from config import PLATFORM_URL, PLATFORM_ADMIN_USER, PLATFORM_ADMIN_PASS, SERVICE_URL

logger = logging.getLogger("business.register")
router = APIRouter(prefix="/api/v1", tags=["注册"])

PLATFORM_API_BASE = f"{PLATFORM_URL}/api/v1"

# 注册两个菜单：一个新标签页跳转，一个内嵌 iframe
MENUS = [
    {
        "code": "external:business:newtab",
        "name": "业务系统（新标签）",
        "icon": "ShopOutlined",
        "menu_type": "menu",
        "sort_order": 10,
    },
    {
        "code": "external:business:iframe",
        "name": "业务系统（内嵌）",
        "icon": "GlobalOutlined",
        "menu_type": "iframe",
        "sort_order": 11,
    },
]


async def _get_admin_token() -> str:
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.post(
            f"{PLATFORM_API_BASE}/accounts/login/",
            json={"username": PLATFORM_ADMIN_USER, "password": PLATFORM_ADMIN_PASS},
        )
        body = resp.json()
        token = body.get("data", {}).get("access")
        if not token:
            raise RuntimeError(f"登录平台失败: {body.get('msg', body)}")
        return token


def _build_menu_data(menu: dict) -> dict:
    return {
        "code": menu["code"],
        "name": menu["name"],
        "icon": menu.get("icon", "LinkOutlined"),
        "path": SERVICE_URL + "/",
        "component": "",
        "permission_code": "",
        "menu_type": menu["menu_type"],
        "is_active": True,
        "is_visible": True,
        "sort_order": menu.get("sort_order", 10),
        "allowed_paths": (
            '["GET:/api/v1/posts/", "POST:/api/v1/posts/",'
            ' "GET:/api/v1/posts/*", "PUT:/api/v1/posts/*",'
            ' "DELETE:/api/v1/posts/*"]'
        ),
    }


async def register_menu() -> dict:
    """幂等注册所有菜单：使用平台 /api/v1/menu/register/ 端点（按 code upsert）"""
    try:
        token = await _get_admin_token()
    except RuntimeError as e:
        logger.error("注册失败: %s", e)
        return {"code": 502, "msg": f"登录平台失败: {e}"}

    headers = {"Authorization": f"Bearer {token}"}
    results = []

    async with httpx.AsyncClient(timeout=10) as client:
        for menu in MENUS:
            data = _build_menu_data(menu)
            resp = await client.post(
                f"{PLATFORM_API_BASE}/menu/register/",
                json=data,
                headers=headers,
            )
            body = resp.json()
            if resp.status_code >= 400 or body.get("code", 200) >= 400:
                logger.error("菜单注册失败 [%s]: %s", menu["code"], body)
                return {"code": resp.status_code, "msg": f"菜单注册失败 [{menu['code']}]: {body.get('msg', str(body))}"}
            logger.info("菜单注册成功: %s (%s)", menu["name"], menu["menu_type"])
            results.append(body.get("data"))

    return {"code": 200, "msg": f"注册成功，共 {len(results)} 个菜单", "data": results}


async def unregister_menu() -> dict:
    """删除所有已注册的菜单"""
    try:
        token = await _get_admin_token()
    except RuntimeError as e:
        return {"code": 502, "msg": f"登录平台失败: {e}"}

    headers = {"Authorization": f"Bearer {token}"}
    deleted = 0

    async with httpx.AsyncClient(timeout=10) as client:
        for menu in MENUS:
            list_resp = await client.get(
                f"{PLATFORM_API_BASE}/menu/",
                params={"search": menu["code"]},
                headers=headers,
            )
            raw = list_resp.json().get("data", [])
            existing = next((m for m in raw if m.get("code") == menu["code"]), None)
            if not existing:
                continue
            del_resp = await client.delete(
                f"{PLATFORM_API_BASE}/menu/{existing['id']}/",
                headers=headers,
            )
            if del_resp.status_code >= 400:
                logger.warning("删除菜单失败 [%s]: %s", menu["code"], del_resp.text)
            else:
                deleted += 1
                logger.info("菜单已删除: %s", menu["code"])

    return {"code": 200, "msg": f"已注销 {deleted} 个菜单"}


@router.post("/register/")
async def register_endpoint():
    result = await register_menu()
    if result.get("code", 200) >= 400:
        raise HTTPException(status_code=result["code"], detail=result["msg"])
    return result


@router.post("/unregister/")
async def unregister_endpoint():
    result = await unregister_menu()
    if result.get("code", 200) >= 400:
        raise HTTPException(status_code=result["code"], detail=result["msg"])
    return result
