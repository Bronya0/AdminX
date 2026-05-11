"""
自定义过滤器

最佳实践:
  1. 继承 django_filters.FilterSet
  2. 用 Meta.fields 声明可筛字段
  3. 复杂查询用 method 参数自定义
"""
import django_filters
from django.db.models import Q

from .models import Post


class PostFilter(django_filters.FilterSet):
    """文章过滤器"""

    # 时间范围筛选
    created_after = django_filters.DateTimeFilter(field_name="created_at", lookup_expr="gte",
                                                   help_text="创建时间起始")
    created_before = django_filters.DateTimeFilter(field_name="created_at", lookup_expr="lte",
                                                    help_text="创建时间截止")

    # 多字段搜索
    keyword = django_filters.CharFilter(method="filter_keyword", help_text="关键词（标题/内容搜索）")

    # 分类支持多个
    category = django_filters.UUIDFilter(field_name="category__id")

    class Meta:
        model = Post
        fields = {
            "status": ["exact"],
            "author": ["exact"],
            "category": ["exact", "isnull"],
        }

    @staticmethod
    def filter_keyword(queryset, name, value):
        """关键词同时搜索标题和内容"""
        if not value:
            return queryset
        return queryset.filter(
            Q(title__icontains=value) | Q(content__icontains=value)
        )