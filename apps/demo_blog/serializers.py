"""
序列化器 — 规范示例

最佳实践:
  1. 用 ModelSerializer 减少样板代码
  2. 显式声明 read_only_fields
  3. 嵌套序列化展示关联数据
  4. validate_xxx 方法做字段级校验
  5. validate 方法做跨字段校验
  6. 给 fields 添加 verbose_name 作为 help_text（API 文档友好）
"""
from rest_framework import serializers

from .models import Category, Post


class CategorySerializer(serializers.ModelSerializer):
    """分类序列化器"""

    # 计算字段：该分类下的文章数（只读）
    post_count = serializers.IntegerField(read_only=True, help_text="该分类下的文章数")

    class Meta:
        model = Category
        fields = "__all__"
        read_only_fields = ["id", "created_at"]


class CategoryListSerializer(serializers.ModelSerializer):
    """
    分类列表序列化器 — 精简版（只返回列表所需字段）
    分开列表和详情序列化器可以避免 N+1 查询
    """
    class Meta:
        model = Category
        fields = ["id", "name", "sort_order", "is_active"]


class PostSerializer(serializers.ModelSerializer):
    """
    文章序列化器

    显式声明嵌套字段而非用 depth，更可控。
    """
    # 关联对象展开显示（只读）
    category_name = serializers.CharField(source="category.name", read_only=True, help_text="分类名称")
    author_name = serializers.CharField(source="author.username", read_only=True, help_text="作者用户名")

    # 标签列表（读写 — 入库存逗号分隔，接口返回数组）
    tag_list = serializers.ListField(
        child=serializers.CharField(),
        write_only=True,
        required=False,
        help_text="标签数组，如 ['python', 'django']",
    )

    class Meta:
        model = Post
        fields = [
            "id", "title", "content", "summary",
            "category", "category_name",
            "author", "author_name",
            "status", "tags", "tag_list",
            "view_count",
            "created_at", "updated_at", "published_at",
        ]
        read_only_fields = ["id", "view_count", "created_at", "updated_at", "published_at"]

    # ── 字段级校验 ──────────────────────────────────

    def validate_title(self, value):
        """标题不能纯空格，长度 2~255"""
        stripped = value.strip()
        if len(stripped) < 2:
            raise serializers.ValidationError("标题至少 2 个字符")
        return stripped

    def validate_tags(self, value):
        """标签去重、修剪空格"""
        if not value.strip():
            return ""
        parts = [t.strip() for t in value.split(",") if t.strip()]
        # 去重但保持顺序
        seen = set()
        unique = []
        for p in parts:
            if p.lower() not in seen:
                seen.add(p.lower())
                unique.append(p)
        return ",".join(unique)

    # ── 跨字段校验 ──────────────────────────────────

    def validate(self, attrs):
        """
        发布时必须填写分类和摘要。
        validate 的 attrs 已经过字段级校验，
        可通过 self.instance 访问已有数据（更新场景）。
        """
        status = attrs.get("status") or getattr(self.instance, "status", None)
        category = attrs.get("category") or getattr(self.instance, "category", None)
        summary = attrs.get("summary") or getattr(self.instance, "summary", "")

        if status == Post.STATUS_PUBLISHED:
            if not category:
                raise serializers.ValidationError({"category": "发布状态必须选择分类"})
            if not summary:
                raise serializers.ValidationError({"summary": "发布状态必须填写摘要"})

        return attrs

    # ── 创建/更新钩子 ───────────────────────────────

    def create(self, validated_data):
        """创建时处理 tag_list 到 tags 的转换"""
        tag_list = validated_data.pop("tag_list", None)
        if tag_list is not None:
            validated_data["tags"] = ",".join(tag_list)
        return super().create(validated_data)

    def update(self, instance, validated_data):
        """更新时处理 tag_list 到 tags 的转换"""
        tag_list = validated_data.pop("tag_list", None)
        if tag_list is not None:
            validated_data["tags"] = ",".join(tag_list)
        return super().update(instance, validated_data)