"""FastAPI 业务服务 — 示例三方业务对接"""

import asyncio
import logging
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, HTMLResponse
from fastapi.staticfiles import StaticFiles

from config import SERVICE_PORT, CORS_ORIGINS, DEBUG, LOG_LEVEL
from routers import posts, register
from routers import component as comp_module
from auth import introspect_token

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL.upper(), logging.INFO),
    format="%(asctime)s [%(name)s] %(levelname)s %(message)s",
)
logger = logging.getLogger("business")

HERE = Path(__file__).resolve().parent

_heartbeat_task: asyncio.Task | None = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    global _heartbeat_task
    # 注册菜单（原有逻辑）
    logger.info("正在注册菜单到平台 %s ...", register.PLATFORM_URL)
    result = await register.register_menu()
    logger.info("注册结果: %s", result.get("msg", result))

    # 注册组件 + 启动心跳
    await comp_module.register()
    _heartbeat_task = asyncio.create_task(comp_module.heartbeat_loop())

    yield

    # 关闭时注销
    if _heartbeat_task:
        _heartbeat_task.cancel()
    await comp_module.unregister()



app = FastAPI(
    title="业务服务",
    description="三方业务对接示例 — 注册菜单 + JWT introspection 鉴权",
    version="1.0.0",
    # 生产环境关闭 Swagger 文档（避免暴露接口结构）
    docs_url="/docs" if DEBUG else None,
    redoc_url="/redoc" if DEBUG else None,
    lifespan=lifespan,
)

# ─── CORS ───
if CORS_ORIGINS == "*":
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=False,  # * 和 credentials 不兼容
        allow_methods=["*"],
        allow_headers=["*"],
    )
else:
    origins = [o.strip() for o in CORS_ORIGINS.split(",")]
    app.add_middleware(
        CORSMiddleware,
        allow_origins=origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

# ─── JWT introspection 中间件 ───
# 拦截 /api/v1/posts/* 请求，提取 Authorization header，
# 转发给 AdminX introspect 接口校验，
# 校验通过后在 request.state.user 注入用户信息。
# 注意: /api/v1/register、/api/v1/unregister 不在此列——
# 它们使用平台管理员账号自行登录平台，无需业务 JWT。
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
    uvicorn.run("main:app", host="0.0.0.0", port=SERVICE_PORT, reload=False)
