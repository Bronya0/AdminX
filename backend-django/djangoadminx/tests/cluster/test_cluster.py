"""cluster — 集群节点"""

from djangoadminx.tests.base import AdminXTestCase
from djangoadminx.tests.factories import UserFactory


class TestCluster(AdminXTestCase):
    def setUp(self):
        self.admin = UserFactory.create_admin()
        self.auth(self.admin)

    def test_list_nodes(self):
        resp = self.client.get("/api/v1/cluster/nodes/")
        self.assert_ok(resp)

    def test_overview(self):
        resp = self.client.get("/api/v1/cluster/nodes/overview/")
        data = self.assert_ok(resp)
        assert "total" in data
        assert "online" in data

    def test_create_node(self):
        resp = self.client.post("/api/v1/cluster/nodes/", {
            "name": "node1", "host": "192.168.1.1", "port": 8000,
        })
        self.assert_created(resp)

    def test_delete_node(self):
        from djangoadminx.cluster.models import ClusterNode
        node = ClusterNode.objects.create(name="del_node", host="10.0.0.1", port=8000)
        resp = self.client.delete(f"/api/v1/cluster/nodes/{node.id}/")
        self.assert_no_content(resp)
