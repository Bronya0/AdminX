import mimetypes

from django.conf import settings
from rest_framework import parsers, serializers, viewsets
from rest_framework.permissions import IsAuthenticated
from rest_framework.response import Response
from rest_framework.views import APIView

from .models import FileRecord, get_storage_backend

try:
    import magic
    HAS_MAGIC = True
except ImportError:
    HAS_MAGIC = False


ALLOWED_MIME_TYPES = {
    "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
    "application/pdf",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/zip",
    "text/plain", "text/csv", "text/xml", "text/json", "application/json",
    # 升级包
    "application/x-tar", "application/gzip", "application/x-gzip",
    "application/x-bzip2", "application/x-xz",
}
MAX_FILE_SIZE = 100 * 1024 * 1024  # 100MB


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

        # 大小校验
        if file_obj.size > MAX_FILE_SIZE:
            return Response({"code": 413, "msg": f"文件过大，最大允许 {MAX_FILE_SIZE // 1024 // 1024}MB"})

        # MIME 类型校验
        file_bytes = file_obj.read(2048)
        file_obj.seek(0)
        if HAS_MAGIC:
            mime_type = magic.from_buffer(file_bytes, mime=True)
        else:
            mime_type = file_obj.content_type or mimetypes.guess_type(file_obj.name)[0] or "application/octet-stream"

        if mime_type not in ALLOWED_MIME_TYPES:
            return Response({"code": 415, "msg": f"不支持的文件类型: {mime_type}"})

        backend_name = getattr(settings, "FILE_STORAGE_BACKEND", "local")
        backend = get_storage_backend(backend_name)
        storage_path = backend.save(file_obj, file_obj.name)

        record = FileRecord.objects.create(
            original_name=file_obj.name,
            size=file_obj.size,
            mime_type=mime_type,
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
