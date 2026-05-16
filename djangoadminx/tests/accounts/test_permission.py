from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory
from djangoadminx.accounts.models import BusinessPermission, BusinessCommand, Role
from djangoadminx.menu.models import Menu


class TestRBACPermission(AdminXTestCase):
    """RBACPermission 正确性 — 路径白名单权限校验"""

    @classmethod
    def setUpTestData(cls):
        # 菜单 A: 无 method 限定的通配规则
        cls.menu_a = Menu.objects.create(
            code="test:users",
            name="用户管理",
            path="/test/users",
            menu_type="menu", depth=1, numchild=0,
            allowed_paths='["/api/v1/accounts/users/*"]',
        )
        # 菜单 B: method + 多规则 + ? 通配符
        cls.menu_b = Menu.objects.create(
            code="test:roles",
            name="角色管理",
            path="/test/roles",
            menu_type="menu", depth=1, numchild=0,
            allowed_paths='["GET:/api/v1/accounts/roles/*", "POST:/api/v1/accounts/roles/"]',
        )
        # 菜单 C: 多个不同的路径
        cls.menu_c = Menu.objects.create(
            code="test:config",
            name="配置管理",
            path="/test/config",
            menu_type="menu", depth=1, numchild=0,
            allowed_paths='["/api/v1/accounts/login-logs/*"]',
        )
        # 业务命令（模拟三方容器）
        cls.biz_cmd = BusinessCommand.objects.create(
            app_label="test_biz",
            name="三方插件",
            menu_path="/test/biz",
            allowed_paths='["GET:/api/v1/notification/*"]',
            is_active=True,
        )
        cls.menu_d = Menu.objects.create(
            code="test:biz",
            name="三方菜单",
            path="/test/biz",
            menu_type="menu", depth=1, numchild=0,
        )

        # 角色 A: 只有菜单 A
        cls.role_a = Role.objects.create(name="角色A", code="role_a")
        cls.role_a.menus.add(cls.menu_a)

        # 角色 B: 菜单 B + C
        cls.role_b = Role.objects.create(name="角色B", code="role_b")
        cls.role_b.menus.add(cls.menu_b, cls.menu_c)

        # 角色 C: 菜单 D（BusinessCommand 关联）
        cls.role_c = Role.objects.create(name="角色C", code="role_c")
        cls.role_c.menus.add(cls.menu_d)

    # ─── 基础场景 ───

    def test_user_view_own_info(self):
        """普通用户访问 /users/me/ 不需要特殊权限"""
        user = UserFactory.create_user()
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/users/me/")
        self.assert_ok(resp)

    def test_admin_can_access_anything(self):
        """超级管理员放行所有"""
        admin = UserFactory.create_admin()
        self.auth(admin)
        resp = self.client.get("/api/v1/accounts/users/")
        self.assert_ok(resp)
        resp = self.client.get("/api/v1/config/")
        self.assert_ok(resp)

    def test_no_role_denied(self):
        """没有角色的用户 403"""
        user = UserFactory.create_user()
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/users/")
        self.assert_fail(resp, 403)

    # ─── 通配符 * 匹配 ───

    def test_wildcard_match(self):
        """* 匹配任意子路径"""
        user = UserFactory.create_user(username="perm_a"); user.roles.add(self.role_a)
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/users/")
        self.assert_ok(resp)

    def test_wildcard_match_sub_path(self):
        """* 匹配多级路径 /users/{id}/"""
        user = UserFactory.create_user(username="perm_a"); user.roles.add(self.role_a)
        self.auth(user)
        target = UserFactory.create_user("target")
        resp = self.client.get(f"/api/v1/accounts/users/{target.id}/")
        self.assert_ok(resp)

    def test_wildcard_match_post(self):
        """无 method 前缀 → 不限 method，POST 放行"""
        user = UserFactory.create_user(username="perm_a"); user.roles.add(self.role_a)
        self.auth(user)
        resp = self.client.post("/api/v1/accounts/users/", {
            "username": "newguy", "password": "test123",
        })
        self.assert_created(resp)

    # ─── method 前缀限制 ───

    def test_method_prefix_allows_get(self):
        """GET: 前缀 → GET 放行"""
        user = UserFactory.create_user(username="perm_b"); user.roles.add(self.role_b)
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/roles/")
        self.assert_ok(resp)

    def test_method_prefix_denies_delete(self):
        """只有 GET 和 POST 规则 → DELETE 拒绝"""
        user = UserFactory.create_user(username="perm_b"); user.roles.add(self.role_b)
        self.auth(user)
        from djangoadminx.accounts.models import Role as RoleModel
        role = RoleModel.objects.create(name="del_role", code="del_role")
        resp = self.client.delete(f"/api/v1/accounts/roles/{role.id}/")
        self.assert_fail(resp, 403)

    # ─── method 混合规则 ───

    def test_method_mixed_rules(self):
        """method 限定与不限定的规则同时存在时各自生效"""
        user = UserFactory.create_user(username="perm_b"); user.roles.add(self.role_b)
        self.auth(user)
        # POST:/api/v1/accounts/roles/ 放行
        resp = self.client.post("/api/v1/accounts/roles/", {"name": "x", "code": "x"})
        self.assert_created(resp)
        # GET:/api/v1/accounts/roles/* 放行
        resp = self.client.get("/api/v1/accounts/roles/")
        self.assert_ok(resp)
        # DELETE 不在任何规则内 → 拒绝
        from djangoadminx.accounts.models import Role as RoleModel
        role = RoleModel.objects.create(name="y", code="y")
        resp = self.client.delete(f"/api/v1/accounts/roles/{role.id}/")
        self.assert_fail(resp, 403)

    # ─── 多规则、多菜单联合 ───

    def test_multiple_rules_both_match(self):
        """菜单 C 有多条路径规则，均放行"""
        user = UserFactory.create_user(username="perm_b"); user.roles.add(self.role_b)
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/login-logs/")
        self.assert_ok(resp)

    def test_path_outside_all_rules_denied(self):
        """多规则都不匹配时拒绝"""
        user = UserFactory.create_user(username="perm_b"); user.roles.add(self.role_b)
        self.auth(user)
        resp = self.client.get("/api/v1/notification/")
        self.assert_fail(resp, 403)

    # ─── 多角色联合 ───

    def test_multiple_roles_union(self):
        """用户有多个角色时，权限取所有角色绑定的菜单并集"""
        user = UserFactory.create_user(username="multi")
        user.roles.add(self.role_a, self.role_b)
        self.auth(user)
        # role_a 有 /api/v1/accounts/users/* 权限
        resp = self.client.get("/api/v1/accounts/users/")
        self.assert_ok(resp)
        # role_b 有 GET:/api/v1/accounts/roles/* 权限
        resp = self.client.get("/api/v1/accounts/roles/")
        self.assert_ok(resp)

    def test_multi_role_one_restricted_still_denied(self):
        """多角色各自有不同权限，某个 URL 在所有角色的规则中都找不到时拒绝"""
        user = UserFactory.create_user(username="multi2")
        user.roles.add(self.role_a, self.role_b)
        self.auth(user)
        # role_a 和 role_b 都没有 webservice 权限
        resp = self.client.get("/api/v1/webservice/jobs/")
        self.assert_fail(resp, 403)

    def test_multi_role_inherit_menu_c(self):
        """角色 A + 角色 B 组合可以访问角色 B 的菜单 C 路径"""
        user = UserFactory.create_user(username="multi3")
        user.roles.add(self.role_b)  # role_b 有 menu_c
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/login-logs/")
        self.assert_ok(resp)

    # ─── BusinessCommand 白名单 ───

    def test_business_command_allowed(self):
        """BusinessCommand 路径白名单同样生效"""
        user = UserFactory.create_user(username="perm_c"); user.roles.add(self.role_c)
        self.auth(user)
        resp = self.client.get("/api/v1/notification/")
        self.assert_ok(resp)

    def test_business_command_method_restricted(self):
        """BusinessCommand 中 GET: 限定 → POST 拒绝"""
        user = UserFactory.create_user(username="perm_c"); user.roles.add(self.role_c)
        self.auth(user)
        resp = self.client.post("/api/v1/notification/")
        self.assert_fail(resp, 403)

    # ─── BusinessPermission 不影响后端 ───

    def test_biz_perm_without_menu_denied(self):
        """有 BusinessPermission 但无菜单绑定 → API 拒绝（仅前端鉴权）"""
        user = UserFactory.create_user(username="only_perm")
        bp = BusinessPermission.objects.create(
            app_label="test", codename="test:only:perm", name="仅权限",
        )
        role = Role.objects.create(name="仅权限角色", code="only_perm_role")
        role.business_permissions.add(bp)
        user.roles.add(role)
        self.auth(user)
        resp = self.client.get("/api/v1/accounts/users/")
        self.assert_fail(resp, 403)



