#!/usr/bin/env python3
"""
DjangoAdminX 轻量反向代理 — 替代 Nginx
依赖: pip install aiohttp
用法: python proxy.py [--port 80] [--backend http://127.0.0.1:9999]
"""
import argparse
import asyncio
import mimetypes
from pathlib import Path

import aiohttp
from aiohttp import web

BASE_DIR = Path(__file__).parent
FRONTEND_DIST = BASE_DIR / "adminx-ui" / "dist"
STATIC_DIR    = BASE_DIR / "staticfiles"
MEDIA_DIR     = BASE_DIR / "media"

# 不转发给后端的 hop-by-hop 头
_HOP_HEADERS = frozenset([
    "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
    "te", "trailers", "transfer-encoding", "upgrade",
])


async def proxy_http(request: web.Request, backend_url: str) -> web.StreamResponse:
    headers = {k: v for k, v in request.headers.items()
               if k.lower() not in _HOP_HEADERS and k.lower() != "host"}
    headers["X-Real-IP"] = request.remote or ""
    headers["X-Forwarded-For"] = request.headers.get("X-Forwarded-For", request.remote or "")
    headers["X-Forwarded-Proto"] = "https"  # 告知 Django 勿做 HTTP→HTTPS 重定向

    async with aiohttp.ClientSession() as session:
        async with session.request(
            method=request.method,
            url=backend_url,
            headers=headers,
            data=await request.read(),
            allow_redirects=False,
        ) as resp:
            out_headers = {k: v for k, v in resp.headers.items()
                           if k.lower() not in _HOP_HEADERS}
            return web.Response(
                status=resp.status,
                headers=out_headers,
                body=await resp.read(),
            )


async def proxy_ws(request: web.Request, backend_url: str) -> web.WebSocketResponse:
    ws = web.WebSocketResponse()
    await ws.prepare(request)

    async with aiohttp.ClientSession() as session:
        async with session.ws_connect(backend_url) as upstream:

            async def fwd_up():
                async for msg in ws:
                    if msg.type == aiohttp.WSMsgType.TEXT:
                        await upstream.send_str(msg.data)
                    elif msg.type == aiohttp.WSMsgType.BINARY:
                        await upstream.send_bytes(msg.data)
                    else:
                        break

            async def fwd_down():
                async for msg in upstream:
                    if msg.type == aiohttp.WSMsgType.TEXT:
                        await ws.send_str(msg.data)
                    elif msg.type == aiohttp.WSMsgType.BINARY:
                        await ws.send_bytes(msg.data)
                    else:
                        break

            await asyncio.gather(fwd_up(), fwd_down())
    return ws


async def serve_file(path: Path, cache_days: int = 7) -> web.Response:
    if not path.is_file():
        raise web.HTTPNotFound()
    mime, _ = mimetypes.guess_type(str(path))
    return web.Response(
        body=path.read_bytes(),
        content_type=mime or "application/octet-stream",
        headers={"Cache-Control": f"public, max-age={cache_days * 86400}"},
    )


def make_handler(backend: str):
    async def handle(request: web.Request) -> web.Response:
        path = request.path
        qs   = ("?" + request.query_string) if request.query_string else ""

        if path == "/djangoadminx":
            raise web.HTTPMovedPermanently("/djangoadminx/")

        # 前端静态资源
        if path.startswith("/djangoadminx/assets/"):
            return await serve_file(FRONTEND_DIST / "assets" / path[21:])

        if path == "/djangoadminx/favicon.ico":
            return await serve_file(FRONTEND_DIST / "favicon.ico")

        # Django 静态 / 媒体文件
        if path.startswith("/djangoadminx/static/"):
            return await serve_file(STATIC_DIR / path[21:])

        if path.startswith("/djangoadminx/media/"):
            return await serve_file(MEDIA_DIR / path[20:], cache_days=30)

        # 健康检查
        if path == "/djangoadminx/health/":
            return await proxy_http(request, f"{backend}/api/v1/common/health/")

        # WebSocket
        if path.startswith("/djangoadminx/ws/"):
            ws_backend = backend.replace("http://", "ws://").replace("https://", "wss://")
            return await proxy_ws(request, f"{ws_backend}/ws/{path[17:]}")

        # API
        if path.startswith("/djangoadminx/api/"):
            return await proxy_http(request, f"{backend}/api/{path[18:]}{qs}")

        # Django Admin
        if path.startswith("/djangoadminx/admin/"):
            return await proxy_http(request, f"{backend}/admin/{path[20:]}{qs}")

        # Vue SPA fallback
        if path.startswith("/djangoadminx/"):
            index = FRONTEND_DIST / "index.html"
            if not index.is_file():
                raise web.HTTPNotFound(reason="前端未构建，请先运行 npm run build")
            return web.Response(body=index.read_bytes(), content_type="text/html")

        raise web.HTTPNotFound()

    return handle


def main():
    parser = argparse.ArgumentParser(description="DjangoAdminX 轻量反向代理")
    parser.add_argument("--port",    type=int, default=80,                       help="监听端口 (默认 80)")
    parser.add_argument("--backend", default="http://127.0.0.1:9999",            help="Django 后端地址")
    parser.add_argument("--host",    default="0.0.0.0",                          help="监听地址 (默认 0.0.0.0)")
    args = parser.parse_args()

    app = web.Application()
    app.router.add_route("*", "/{path_info:.*}", make_handler(args.backend))

    print(f"代理启动: http://{args.host}:{args.port}/djangoadminx/")
    print(f"后端地址: {args.backend}")
    web.run_app(app, host=args.host, port=args.port, access_log=None)


if __name__ == "__main__":
    main()
