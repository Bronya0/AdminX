"""
WSDL 发布 — 对外暴露 SOAP 接口供第三方调用

最佳实践:
  1. 每个服务继承 spyne.ServiceBase，用 @srpc 声明方法
  2. 方法与业务代码分离，只做数据转换和编排
  3. 通过 _make_soap_view 工厂函数创建 Django 视图
  4. JWT 认证保护（复用框架的认证机制）
  5. 返回简单字符串格式，避免复杂 XML 类型
"""
import logging

from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from spyne import Application, srpc, ServiceBase
from spyne.model.complex import Iterable
from spyne.model.primitive import Integer, String
from spyne.protocol.soap import Soap11
from spyne.server.django import DjangoView

logger = logging.getLogger("apps.demo_blog.publish")


class BlogPostService(ServiceBase):
    """
    博客文章查询服务 — 对外 SOAP 接口
    WSDL 地址: /api/v1/demo/publish/post/?wsdl
    """

    @srpc(String, _returns=Iterable(String))
    def list_posts(keyword=""):
        """
        获取文章列表

        :param keyword: 可选搜索关键词
        :return: 每篇文章的管道符分隔格式
        """
        # 函数内延迟导入，避免模块加载时的循环依赖
        from .models import Post

        qs = Post.objects.select_related("category", "author").all()
        if keyword:
            qs = qs.filter(title__icontains=keyword)

        for post in qs.order_by("-created_at")[:50]:
            yield (
                f"id={post.id}|title={post.title}|"
                f"status={post.status}|"
                f"category={post.category.name if post.category else ''}|"
                f"author={post.author.username if post.author else ''}|"
                f"view_count={post.view_count}|"
                f"created_at={post.created_at.isoformat() if post.created_at else ''}"
            )

    @srpc(String, _returns=String)
    def get_post(post_id=""):
        """
        根据 ID 获取单篇文章详情

        :param post_id: 文章 UUID
        :return: 管道符分隔格式，不存在返回空字符串
        """
        from .models import Post

        try:
            post = Post.objects.select_related("category", "author").get(id=post_id)
            return (
                f"id={post.id}|title={post.title}|"
                f"content={post.content[:500]}|"
                f"status={post.status}|"
                f"category={post.category.name if post.category else ''}|"
                f"author={post.author.username if post.author else ''}|"
                f"tags={post.tags}|view_count={post.view_count}|"
                f"created_at={post.created_at.isoformat() if post.created_at else ''}|"
                f"updated_at={post.updated_at.isoformat() if post.updated_at else ''}"
            )
        except Post.DoesNotExist:
            return ""

    @srpc(_returns=String)
    def get_statistics():
        """
        获取文章统计（标题和内容搜索）

        :return: JSON 格式的统计字符串
        """
        from .tasks import post_statistics_summary

        import json
        stats = post_statistics_summary()
        return json.dumps(stats, ensure_ascii=False)

    @srpc(String, String, String, _returns=String)
    def create_post(title="", content="", category_id=""):
        """
        创建文章（简单版，供第三方调用）

        :param title: 文章标题
        :param content: 文章内容
        :param category_id: 分类 UUID（可选）
        :return: 创建后的文章 ID，失败返回错误信息
        """
        from .models import Category, Post

        try:
            category = None
            if category_id:
                category = Category.objects.get(id=category_id)

            post = Post.objects.create(
                title=title.strip() if title else "无标题",
                content=content or "",
                category=category,
                status=Post.STATUS_DRAFT,
            )
            logger.info(f"SOAP created post {post.id}: {post.title}")
            return str(post.id)
        except Category.DoesNotExist:
            return f"error: 分类不存在 {category_id}"
        except Exception as e:
            logger.exception("SOAP create_post failed")
            return f"error: {e}"


# ── 构建 spyne Application ──────────────────────

_application = Application(
    [BlogPostService],
    tns="http://djangoadminx.com/soap/demo/blog",
    in_protocol=Soap11(validator="lxml"),
    out_protocol=Soap11(),
    name="BlogPostService",
)


def _auth_check(request):
    """
    JWT 认证检查
    复用框架的 JWTAuthentication，与 API 保持一致的认证方式。
    """
    from rest_framework_simplejwt.authentication import JWTAuthentication

    try:
        auth = JWTAuthentication()
        user, _ = auth.authenticate(request)
        if user is None:
            return None, JsonResponse({"code": 401, "msg": "未授权访问，请提供有效的 JWT token"})
        return user, None
    except Exception:
        return None, JsonResponse({"code": 401, "msg": "未授权访问，请提供有效的 JWT token"})


# ── 创建 Django 视图 ─────────────────────────────

def _make_soap_view():
    """
    工厂函数 — 创建绑定到 BlogPostService 的 Django 视图
    封装了 JWT 认证和 CSRF 豁免。
    """
    django_view = DjangoView.as_view(application=_application)

    @csrf_exempt
    def view(request, **kwargs):
        # JWT 认证
        user, err = _auth_check(request)
        if err:
            return err
        return django_view(request, **kwargs)

    return view


soap_post_view = _make_soap_view()