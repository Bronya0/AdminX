"""FastAPI 业务服务 — 示例三方业务对接"""

import logging
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, HTMLResponse
from fastapi.staticfiles import StaticFiles

from config import SERVICE_PORT, CORS_ORIGINS, DEBUG, LOG_LEVEL
from routers import posts, register
from auth import introspect_token

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL.upper(), logging.INFO),
    format="%(asctime)s [%(name)s] %(levelname)s %(message)s",
)
logger = logging.getLogger("business")

HERE = Path(__file__).resolve().parent


@asynccontextmanager
async def lifespan(app: FastAPI):
    """启动时自动注册菜单（幂等）"""
    logger.info("正在注册菜单到平台 %s ...", register.PLATFORM_URL)
    result = await register.register_menu()
    logger.info("注册结果: %s", result.get("msg", result))
    yield


app = FastAPI(
    title="业务服务",
    description="三方业务对接示例 — 注册菜单 + JWT introspection 鉴权",
    version="1.0.0",
    docs_url="/docs",
    lifespan=lifespan,
)

# ─── CORS ───
origins = [o.strip() for o in CORS_ORIGINS.split(",")] if CORS_ORIGINS != "*" else ["*"]
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# ─── JWT introspection 中间件 ───
# 拦截 /api/v1/posts/* 请求，提取 Authorization header，
# 转发给 DjangoAdminX introspect 接口校验，
# 校验通过后在 request.state.user 注入用户信息。
PROTECTED_PREFIXES = ("/api/v1/posts",)


@app.middleware("http")
async def auth_middleware(request: Request, call_next):
    path = request.url.path

    if any(path.startswith(p) for p in PROTECTED_PREFIXES):
        auth_header = request.headers.get("Authorization", "")
        if not auth_header.startswith("Bearer "):
            return JSONResponse(
                status_code=401,
                content={"code": 401, "msg": "缺少 Authorization 头"},
            )

        token = auth_header.removeprefix("Bearer ").strip()
        try:
            user_info = await introspect_token(token)
        except Exception as e:
            logger.warning("introspect 失败: %s", e)
            return JSONResponse(
                status_code=502,
                content={"code": 502, "msg": "鉴权服务不可达"},
            )
        if user_info is None:
            return JSONResponse(
                status_code=401,
                content={"code": 401, "msg": "Token 无效或已过期"},
            )

        request.state.user = user_info
        logger.debug("JWT 校验通过: %s", user_info.get("username"))

    return await call_next(request)


app.include_router(posts.router)
app.include_router(register.router)

app.mount("/static", StaticFiles(directory=str(HERE / "static")), name="static")


@app.get("/", include_in_schema=False)
async def index():
    content = (HERE / "static" / "index.html").read_text(encoding="utf-8")
    return HTMLResponse(content)


@app.get("/healthz", tags=["系统"])
async def healthz():
    return {"status": "ok", "service": "business-service", "version": "1.0.0"}


if __name__ == "__main__":
    import uvicorn
    logger.info("启动业务服务 http://localhost:%d  (DEBUG=%s)", SERVICE_PORT, DEBUG)
    uvicorn.run("main:app", host="0.0.0.0", port=SERVICE_PORT, reload=DEBUG)
