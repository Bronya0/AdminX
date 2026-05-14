from rest_framework import viewsets
from rest_framework.decorators import action
from rest_framework.permissions import IsAdminUser
from rest_framework.response import Response

from djangoadminx.audit.mixins import AuditLogMixin
from .models import ClusterNode
from .serializers import ClusterNodeSerializer


class ClusterNodeViewSet(AuditLogMixin, viewsets.ModelViewSet):
    """集群节点 CRUD"""
    queryset = ClusterNode.objects.all()
    serializer_class = ClusterNodeSerializer
    permission_classes = [IsAdminUser]
    search_fields = ["name", "host"]
    ordering_fields = ["name", "status", "created_at"]

    @action(detail=False, methods=["get"], permission_classes=[IsAdminUser])
    def overview(self, request):
        """集群概览"""
        total = ClusterNode.objects.count()
        online = ClusterNode.objects.filter(status="online").count()
        offline = ClusterNode.objects.filter(status="offline").count()
        return Response({
            "code": 200,
            "msg": "success",
            "data": {
                "total": total,
                "online": online,
                "offline": offline,
                "nodes": ClusterNodeSerializer(self.get_queryset(), many=True).data,
            },
        })