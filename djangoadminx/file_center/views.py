import mimetypes

from django.conf import settings
from rest_framework import parsers, serializers, viewsets
from rest_framework.permissions import IsAuthenticated
from rest_framework.response import Response
from rest_framework.views import APIView

from .models import FileRecord, get_storage_backend


class FileUploadSerializer(serializers.Serializer):
    file = serializers.FileField()


class FileRecordSerializer(serializers.ModelSerializer):
    class Meta:
        model = FileRecord
        fields = "__all__"


class FileUploadView(APIView):
    """文件上传"""
    permission_classes = [IsAuthenticated]

    def post(self, request):
        ser = FileUploadSerializer(data=request.data)
        ser.is_valid(raise_exception=True)

        file_obj = ser.validated_data["file"]
        backend_name = getattr(settings, "FILE_STORAGE_BACKEND", "local")
        backend = get_storage_backend(backend_name)

        storage_path = backend.save(file_obj, file_obj.name)

        record = FileRecord.objects.create(
            original_name=file_obj.name,
            size=file_obj.size,
            mime_type=file_obj.content_type or mimetypes.guess_type(file_obj.name)[0] or "application/octet-stream",
            storage_backend=backend_name,
            storage_path=storage_path,
            url=backend.url(storage_path),
            uploaded_by=request.user.username,
        )

        return Response({
            "code": 200,
            "msg": "success",
            "data": FileRecordSerializer(record).data,
        })


class FileViewSet(viewsets.ReadOnlyModelViewSet):
    """文件记录列表"""
    queryset = FileRecord.objects.all()
    serializer_class = FileRecordSerializer
    permission_classes = [IsAuthenticated]
    ordering = ["-created_at"]