"""
单元测试 — 规范示例

最佳实践:
  1. 继承 APITestCase，使用 Django 的测试客户端
  2. 每个测试方法只测一件事
  3. 方法名以 test_ 开头，清晰描述测试场景
  4. 使用 setUpTestData 准备共享数据
  5. 测试边界条件（空值、非法输入、权限不足）

注意: StandardJsonRenderer 将 all 响应包装为 {code, msg, data}
      并强制 status_code=200，因此测试需 render() 后解析 JSON。
"""
import json

from django.contrib.auth import get_user_model
from rest_framework import status
from rest_framework.test import APITestCase

from .models import Category, Post

User = get_user_model()


def _json(response):
    """render 并解析 JSON 响应"""
    response.render()
    return json.loads(response.content)


class CategoryTests(APITestCase):
    """分类 API 测试"""

    @classmethod
    def setUpTestData(cls):
        """准备测试数据（在所有测试方法前执行一次）"""
        cls.user = User.objects.create_user(username="testuser", password="testpass123")
        cls.category = Category.objects.create(name="Python", sort_order=1)

    def setUp(self):
        """每个测试方法前执行"""
        self.client.force_authenticate(user=self.user)

    # ── 列表 ──

    def test_list_categories_success(self):
        """GET /api/v1/demo/categories/ → 返回分类列表"""
        response = self.client.get("/api/v1/demo/categories/")
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        data = _json(response)
        self.assertEqual(data["code"], 200)
        self.assertGreaterEqual(len(data["data"]["results"]), 1)

    # ── 创建 ──

    def test_create_category_success(self):
        """POST /api/v1/demo/categories/ → 正常创建"""
        data = {"name": "JavaScript", "sort_order": 2}
        response = self.client.post("/api/v1/demo/categories/", data)
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        data = _json(response)
        self.assertEqual(data["code"], 201)

    def test_create_category_duplicate_name(self):
        """重复分类名称 → 400"""
        data = {"name": "Python"}
        response = self.client.post("/api/v1/demo/categories/", data)
        data = _json(response)
        self.assertEqual(data["code"], 400)

    # ── 更新 ──

    def test_update_category_success(self):
        """PUT /api/v1/demo/categories/{id}/ → 正常更新"""
        data = {"name": "Python 3", "sort_order": 1}
        response = self.client.put(f"/api/v1/demo/categories/{self.category.id}/", data)
        data = _json(response)
        self.assertEqual(data["code"], 200)
        self.category.refresh_from_db()
        self.assertEqual(self.category.name, "Python 3")

    # ── 删除 ──

    def test_delete_category_success(self):
        """DELETE /api/v1/demo/categories/{id}/ → 正常删除"""
        response = self.client.delete(f"/api/v1/demo/categories/{self.category.id}/")
        data = _json(response)
        self.assertEqual(data["code"], 204)
        self.assertFalse(Category.objects.filter(id=self.category.id).exists())


class PostTests(APITestCase):
    """文章 API 测试"""

    @classmethod
    def setUpTestData(cls):
        cls.user = User.objects.create_user(username="author", password="testpass123")
        cls.category = Category.objects.create(name="Django", sort_order=1)
        cls.post = Post.objects.create(
            title="Test Post",
            content="Content here",
            category=cls.category,
            author=cls.user,
            status=Post.STATUS_DRAFT,
        )

    def setUp(self):
        self.client.force_authenticate(user=self.user)

    def test_create_post_auto_author(self):
        """创建文章自动填充 author"""
        data = {"title": "New Post", "content": "Hello"}
        response = self.client.post("/api/v1/demo/posts/", data)
        data = _json(response)
        self.assertEqual(data["code"], 201)
        # 验证 author 是当前登录用户
        post_id = data["data"]["id"]
        post = Post.objects.get(id=post_id)
        self.assertEqual(post.author, self.user)

    def test_publish_draft_success(self):
        """发布草稿 → 状态变为 published + 填充发布时间"""
        response = self.client.post(f"/api/v1/demo/posts/{self.post.id}/publish/")
        data = _json(response)
        self.assertEqual(data["code"], 200)
        self.post.refresh_from_db()
        self.assertEqual(self.post.status, Post.STATUS_PUBLISHED)
        self.assertIsNotNone(self.post.published_at)

    def test_publish_already_published(self):
        """已发布文章再次发布 → 400"""
        self.post.status = Post.STATUS_PUBLISHED
        self.post.save()
        response = self.client.post(f"/api/v1/demo/posts/{self.post.id}/publish/")
        data = _json(response)
        self.assertEqual(data["code"], 400)

    def test_increment_view(self):
        """增加浏览量 → view_count +1"""
        old_count = self.post.view_count
        response = self.client.post(f"/api/v1/demo/posts/{self.post.id}/increment_view/")
        data = _json(response)
        self.assertEqual(data["code"], 200)
        self.post.refresh_from_db()
        self.assertEqual(self.post.view_count, old_count + 1)

    def test_unauthenticated_access(self):
        """未认证用户 → 401"""
        self.client.force_authenticate(user=None)
        response = self.client.get("/api/v1/demo/posts/")
        response.render()
        data = json.loads(response.content)
        self.assertEqual(data["code"], 401)