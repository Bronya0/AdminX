"""菜单注册 — 向 DjangoAdminX 平台注册自身菜单（幂等）"""

import logging

from fastapi import APIRouter, HTTPException
import httpx

from config import PLATFORM_URL, PLATFORM_ADMIN_USER, PLATFORM_ADMIN_PASS, SERVICE_URL

logger = logging.getLogger("business.register")
router = APIRouter(prefix="/api/v1", tags=["注册"])

MENU_CODE = "external:business"

PLATFORM_API_BASE = f"{PLATFORM_URL}/api/v1"


async def _get_admin_token() -> str:
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.post(
            f"{PLATFORM_API_BASE}/accounts/login/",
            json={"username": PLATFORM_ADMIN_USER, "password": PLATFORM_ADMIN_PASS},
        )
        body = resp.json()
        data = body.get("data", {})
        token = data.get("access")
        if not token:
            raise RuntimeError(f"登录平台失败: {body.get('msg', body)}")
        return token


async def _build_menu_data() -> dict:
    return {
        "code": MENU_CODE,
        "name": "业务管理",
        "icon": "ShopOutlined",
        "path": SERVICE_URL + "/",
        "component": "",
        "permission_code": "",
        "menu_type": "menu",
        "is_active": True,
        "is_visible": True,
        "sort_order": 10,
        "allowed_paths": (
            '["GET:/api/v1/posts/", "POST:/api/v1/posts/",'
            ' "GET:/api/v1/posts/*", "PUT:/api/v1/posts/*",'
            ' "DELETE:/api/v1/posts/*"]'
        ),
    }


async def register_menu() -> dict:
    """幂等注册：按 code 查找，存在则更新，不存在则创建"""
    try:
        token = await _get_admin_token()
    except RuntimeError as e:
        logger.error("注册失败: %s", e)
        return {"code": 502, "msg": f"登录平台失败: {e}"}

    headers = {"Authorization": f"Bearer {token}"}
    menu_data = await _build_menu_data()

    async with httpx.AsyncClient(timeout=10) as client:
        # 1. 查询是否已存在
        list_resp = await client.get(
            f"{PLATFORM_API_BASE}/menu/",
            params={"search": MENU_CODE},
            headers=headers,
        )
        list_body = list_resp.json()
        raw = list_body.get("data", list_body.get("results", []))
        existing = raw[0] if isinstance(raw, list) and len(raw) > 0 else None

        if existing:
            menu_id = existing.get("id")
            resp = await client.put(
                f"{PLATFORM_API_BASE}/menu/{menu_id}/",
                json=menu_data,
                headers=headers,
            )
            action = "更新"
        else:
            resp = await client.post(
                f"{PLATFORM_API_BASE}/menu/",
                json=menu_data,
                headers=headers,
            )
            action = "创建"

        body = resp.json()

    if resp.status_code >= 400 or body.get("code", 200) >= 400:
        logger.error("菜单%s失败: %s", action, body)
        return {"code": resp.status_code, "msg": f"菜单{action}失败: {body.get('msg', str(body))}"}

    logger.info("菜单%s成功: %s", action, menu_data["name"])
    return {"code": 200, "msg": f"{action}成功", "data": body.get("data")}


async def unregister_menu() -> dict:
    """删除已注册的菜单"""
    try:
        token = await _get_admin_token()
    except RuntimeError as e:
        return {"code": 502, "msg": f"登录平台失败: {e}"}

    headers = {"Authorization": f"Bearer {token}"}

    async with httpx.AsyncClient(timeout=10) as client:
        list_resp = await client.get(
            f"{PLATFORM_API_BASE}/menu/",
            params={"search": MENU_CODE},
            headers=headers,
        )
        list_body = list_resp.json()
        raw = list_body.get("data", list_body.get("results", []))
        existing = raw[0] if isinstance(raw, list) and len(raw) > 0 else None

        if not existing:
            return {"code": 200, "msg": "菜单不存在，无需删除"}

        menu_id = existing.get("id")
        del_resp = await client.delete(
            f"{PLATFORM_API_BASE}/menu/{menu_id}/",
            headers=headers,
        )
        if del_resp.status_code >= 400:
            logger.warning("删除菜单失败: %s", del_resp.text)
            return {"code": del_resp.status_code, "msg": "删除失败"}

    logger.info("菜单已删除: %s", MENU_CODE)
    return {"code": 200, "msg": "已注销"}


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
