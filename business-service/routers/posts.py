"""文章 CRUD — 示例业务接口，演示 JWT introspection 鉴权"""

from datetime import datetime, timezone

from fastapi import APIRouter, HTTPException, Query, Request

from models import Post, PostCreate, PostUpdate

router = APIRouter(prefix="/api/v1/posts", tags=["文章"])
_db: list[Post] = []
_counter = 0


def _now():
    return datetime.now(timezone.utc)


def _next_id() -> int:
    global _counter
    _counter += 1
    return _counter


def _not_found(post_id: int):
    # 与平台统一 {code, msg, data} 契约
    return HTTPException(
        status_code=404,
        detail={"code": 404, "msg": "文章不存在", "data": None},
    )


def _current_username(request: Request) -> str:
    """introspect 中间件注入的用户名（未登录场景为空，仅用于演示隔离）"""
    user = getattr(request.state, "user", None)
    return (user or {}).get("username", "")


@router.get("/")
async def list_posts(
    request: Request,
    status: str | None = Query(None, description="按状态筛选"),
    search: str | None = Query(None, description="标题关键词搜索"),
    skip: int = Query(0, ge=0, description="跳过条数"),
    limit: int = Query(20, ge=1, le=100, description="每页条数"),
):
    """文章列表（按创建者隔离），支持分页、状态筛选和标题搜索"""
    owner = _current_username(request)
    items = [p for p in _db if p.owner == owner]

    # 筛选
    if status:
        items = [p for p in items if p.status == status]
    if search:
        items = [p for p in items if search.lower() in p.title.lower()]

    # 按创建时间倒序
    items.sort(key=lambda p: p.created_at or datetime.min.replace(tzinfo=timezone.utc), reverse=True)

    total = len(items)
    page = items[skip : skip + limit]

    return {
        "code": 200,
        "msg": "ok",
        "data": {
            "count": total,
            "results": page,
        },
    }


@router.post("/", status_code=201)
async def create_post(body: PostCreate, request: Request):
    """新建文章"""
    post = Post(
        id=_next_id(),
        owner=_current_username(request),
        title=body.title,
        content=body.content,
        status=body.status,
        created_at=_now(),
        updated_at=_now(),
    )
    _db.append(post)
    return {"code": 201, "msg": "创建成功", "data": post}


@router.get("/{post_id}")
async def get_post(post_id: int, request: Request):
    """获取单篇文章（仅本人可见）"""
    owner = _current_username(request)
    for p in _db:
        if p.id == post_id and p.owner == owner:
            return {"code": 200, "msg": "ok", "data": p}
    raise _not_found(post_id)


@router.put("/{post_id}")
async def update_post(post_id: int, body: PostCreate, request: Request):
    """全量更新文章"""
    owner = _current_username(request)
    for i, p in enumerate(_db):
        if p.id == post_id and p.owner == owner:
            _db[i] = Post(
                id=post_id,
                owner=p.owner,
                title=body.title,
                content=body.content,
                status=body.status,
                created_at=p.created_at,
                updated_at=_now(),
            )
            return {"code": 200, "msg": "更新成功", "data": _db[i]}
    raise _not_found(post_id)


@router.patch("/{post_id}")
async def patch_post(post_id: int, body: PostUpdate, request: Request):
    """部分更新文章"""
    owner = _current_username(request)
    for i, p in enumerate(_db):
        if p.id == post_id and p.owner == owner:
            update_data = body.model_dump(exclude_unset=True)
            for field, value in update_data.items():
                setattr(_db[i], field, value)
            _db[i].updated_at = _now()
            return {"code": 200, "msg": "更新成功", "data": _db[i]}
    raise _not_found(post_id)


@router.delete("/{post_id}")
async def delete_post(post_id: int, request: Request):
    """删除文章"""
    owner = _current_username(request)
    for i, p in enumerate(_db):
        if p.id == post_id and p.owner == owner:
            _db.pop(i)
            return {"code": 200, "msg": "已删除"}
    raise _not_found(post_id)
