from django.db import models
from treebeard.mp_tree import MP_Node


class Menu(MP_Node):
    """动态菜单 — 物化路径树形结构"""

    code = models.CharField("菜单编码", max_length=128, unique=True)
    name = models.CharField("菜单名称", max_length=128)
    icon = models.CharField("图标", max_length=64, blank=True, default="")
    path = models.CharField("路由路径", max_length=512, blank=True, default="")
    component = models.CharField("组件路径", max_length=512, blank=True, default="")
    permission_code = models.CharField("权限编码", max_length=256, blank=True, default="",
                                       help_text="如 accounts:user:list，空表示不需要权限")
    menu_type = models.CharField("菜单类型", max_length=20,
                                 choices=[("menu", "菜单"), ("button", "按钮"), ("iframe", "Iframe")],
                                 default="menu")
    is_active = models.BooleanField("启用", default=True)
    is_visible = models.BooleanField("是否可见", default=True)
    sort_order = models.IntegerField("排序", default=0)
    allowed_paths = models.TextField("接口路径白名单", blank=True, default="[]",
                                      help_text='JSON 数组，格式同 BusinessCommand，如 ["GET:/api/v1/menu/*"]')
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "菜单"
        verbose_name_plural = "菜单"
        ordering = ["sort_order"]

    def __str__(self):
        return self.name