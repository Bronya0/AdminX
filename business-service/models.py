"""Pydantic 数据模型"""

from datetime import datetime
from typing import Literal

from pydantic import BaseModel


# ─── 业务模型 ───

class Post(BaseModel):
    id: int | None = None
    title: str
    content: str
    status: str = "draft"
    created_at: datetime | None = None
    updated_at: datetime | None = None


class PostCreate(BaseModel):
    title: str
    content: str
    status: Literal["draft", "published"] = "draft"


class PostUpdate(BaseModel):
    title: str | None = None
    content: str | None = None
    status: Literal["draft", "published"] | None = None


# ─── 统一响应模型（与 DjangoAdminX 平台保持一致） ───

class ApiResponse(BaseModel):
    code: int = 200
    msg: str = "ok"
    data: object | None = None


class PaginatedData(BaseModel):
    count: int
    results: list


class PaginatedResponse(BaseModel):
    code: int = 200
    msg: str = "ok"
    data: PaginatedData
