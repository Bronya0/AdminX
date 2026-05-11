from django.contrib.auth.models import Permission
from django.contrib.contenttypes.models import ContentType
from rest_framework import serializers

from .models import Role, User, UserLoginLog


class UserSerializer(serializers.ModelSerializer):
    roles = serializers.SlugRelatedField(
        many=True, slug_field="code", queryset=Role.objects.all(), required=False
    )
    role_names = serializers.SerializerMethodField()

    class Meta:
        model = User
        fields = [
            "id", "username", "phone", "email", "avatar",
            "is_active", "roles", "role_names",
            "date_joined", "last_login",
        ]
        read_only_fields = ["id", "date_joined", "last_login"]

    def get_role_names(self, obj):
        return [r.name for r in obj.roles.all()]


class UserCreateSerializer(serializers.ModelSerializer):
    password = serializers.CharField(write_only=True, min_length=6)
    roles = serializers.SlugRelatedField(
        many=True, slug_field="code", queryset=Role.objects.all(), required=False
    )

    class Meta:
        model = User
        fields = ["id", "username", "password", "phone", "email", "avatar", "is_active", "roles"]

    def create(self, validated_data):
        roles = validated_data.pop("roles", [])
        user = User.objects.create_user(**validated_data)
        user.roles.set(roles)
        return user


class RoleSerializer(serializers.ModelSerializer):
    permissions = serializers.SlugRelatedField(
        many=True, slug_field="codename", queryset=Permission.objects.all(), required=False
    )
    menus = serializers.SlugRelatedField(
        many=True, slug_field="code", queryset=Role._meta.get_field("menus").remote_field.model.objects.all(), required=False
    )

    class Meta:
        model = Role
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]


class PermissionSerializer(serializers.ModelSerializer):
    content_type_name = serializers.CharField(source="content_type.name", read_only=True)

    class Meta:
        model = Permission
        fields = ["id", "name", "codename", "content_type", "content_type_name"]


class LoginSerializer(serializers.Serializer):
    username = serializers.CharField()
    password = serializers.CharField()
    captcha = serializers.CharField(required=False)


class LoginLogSerializer(serializers.ModelSerializer):
    class Meta:
        model = UserLoginLog
        fields = "__all__"