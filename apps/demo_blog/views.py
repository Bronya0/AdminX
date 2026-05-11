"""
视图 — 规范示例

最佳实践:
  1. 用 viewsets.ModelViewSet 减少重复代码
  2. permission_classes 统一用框架的 RBACPermission
  3. search_fields / ordering_fields / filterset_class 声明查询能力
  4. action 装饰器添加自定义端点
  5. 异常由框架统一处理，视图只关注业务逻辑
  6. 响应统一返回 {code, msg, data} 格式（由 StandardJsonRenderer 自动处理）
"""
import logging

from django.db.models import F
from django.utils import timezone
from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAuthenticated
from rest_framework.response import Response

from djangoadminx.accounts.permissions import RBACPermission

from .filters import PostFilter
from .models import Category, Post
from .serializers import CategorySerializer, PostSerializer

logger = logging.getLogger("apps.demo_blog")


class CategoryViewSet(viewsets.ModelViewSet):
    """
    分类 CRUD

    list   : GET  /api/v1/demo/categories/
    create : POST /api/v1/demo/categories/
    read   : GET  /api/v1/demo/categories/{id}/
    update : PUT  /api/v1/demo/categories/{id}/
    delete : DELETE /api/v1/demo/categories/{id}/
    """
    queryset = Category.objects.all()
    serializer_class = CategorySerializer
    permission_classes = [IsAuthenticated, RBACPermission]

    # ── 查询能力声明 ──
    search_fields = ["name"]
    ordering_fields = ["sort_order", "name", "created_at"]
    ordering = ["sort_order"]

    # ── 额外操作 ──

    @action(detail=True, methods=["post"])
    def toggle_active(self, request, pk=None):
        """切换启用/禁用状态"""
        category = self.get_object()
        category.is_active = not category.is_active
        category.save(update_fields=["is_active"])
        logger.info(f"Category {category.id} toggled to is_active={category.is_active}")
        return Response({"code": 200, "msg": "success", "data": {"is_active": category.is_active}})


class PostViewSet(viewsets.ModelViewSet):
    """
    文章 CRUD + 额外操作

    list    : GET  /api/v1/demo/posts/
    create  : POST /api/v1/demo/posts/
    read    : GET  /api/v1/demo/posts/{id}/
    update  : PUT  /api/v1/demo/posts/{id}/
    partial : PATCH /api/v1/demo/posts/{id}/
    delete  : DELETE /api/v1/demo/posts/{id}/
    publish : POST /api/v1/demo/posts/{id}/publish/
    stats   : GET  /api/v1/demo/posts/stats/
    """
    queryset = Post.objects.select_related("category", "author").all()
    serializer_class = PostSerializer
    permission_classes = [IsAuthenticated, RBACPermission]

    # ── 查询能力声明 ──
    filterset_class = PostFilter
    search_fields = ["title", "content", "tags"]
    ordering_fields = ["title", "created_at", "updated_at", "view_count", "published_at"]
    ordering = ["-created_at"]

    # ── 自动填充当前用户为作者 ──

    def perform_create(self, serializer):
        """创建时自动设置 author 为当前登录用户"""
        serializer.save(author=self.request.user)

    # ── 自定义操作 ──

    @action(detail=True, methods=["post"])
    def publish(self, request, pk=None):
        """发布文章（draft → published）"""
        post = self.get_object()
        if post.status != Post.STATUS_DRAFT:
            return Response({"code": 400, "msg": "仅草稿状态可以发布"})

        post.status = Post.STATUS_PUBLISHED
        post.published_at = timezone.now()
        post.save(update_fields=["status", "published_at"])
        logger.info(f"Post {post.id} published by {request.user.username}")
        return Response({"code": 200, "msg": "发布成功"})

    @action(detail=False, methods=["get"])
    def stats(self, request):
        """
        文章统计
        注意：detail=False 表示这是一个集合操作，不使用 pk
        """
        total = Post.objects.count()
        published = Post.objects.filter(status=Post.STATUS_PUBLISHED).count()
        draft = Post.objects.filter(status=Post.STATUS_DRAFT).count()
        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "total": total,
                "published": published,
                "draft": draft,
                "archived": total - published - draft,
            },
        })

    @action(detail=True, methods=["post"])
    def increment_view(self, request, pk=None):
        """增加浏览量"""
        post = self.get_object()
        Post.objects.filter(id=post.id).update(view_count=F("view_count") + 1)
        # 用 F() 避免并发下的竞态条件，比先读后写更安全
        post.refresh_from_db()
        return Response({
            "code": 200,
            "msg": "success",
            "data": {"view_count": post.view_count},
        })