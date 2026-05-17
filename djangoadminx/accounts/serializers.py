from django.contrib.auth.models import Permission
from django.contrib.contenttypes.models import ContentType
from rest_framework import serializers

from .models import BusinessCommand, BusinessPermission, Role, User, UserLoginLog


class UserSerializer(serializers.ModelSerializer):
    roles = serializers.SlugRelatedField(
        many=True, slug_field="code", queryset=Role.objects.all(), required=False
    )
    role_names = serializers.SerializerMethodField()
    is_online = serializers.ReadOnlyField()

    class Meta:
        model = User
        fields = [
            "id", "username", "phone", "email", "avatar", "desc",
            "is_active", "is_superuser", "roles", "role_names",
            "date_joined", "last_login", "last_activity", "is_online",
        ]
        read_only_fields = ["id", "date_joined", "last_login", "last_activity", "is_online"]

    def get_role_names(self, obj):
        return [r.name for r in obj.roles.all()]


class UserCreateSerializer(serializers.ModelSerializer):
    password = serializers.CharField(write_only=True, min_length=6)
    roles = serializers.SlugRelatedField(
        many=True, slug_field="code", queryset=Role.objects.all(), required=False
    )

    class Meta:
        model = User
        fields = ["id", "username", "password", "phone", "email", "avatar", "desc", "is_active", "roles"]

    def create(self, validated_data):
        from djangoadminx.policy.models import PasswordPolicy
        roles = validated_data.pop("roles", [])
        password = validated_data.pop("password", "")
        policy = PasswordPolicy.get_instance()
        if policy:
            is_valid, errors = policy.validate(password)
            if not is_valid:
                raise serializers.ValidationError({"password": "; ".join(errors)})
        user = User.objects.create_user(**validated_data, password=password)
        user.roles.set(roles)
        return user


class RoleSerializer(serializers.ModelSerializer):
    permissions = serializers.SlugRelatedField(
        many=True, slug_field="codename", queryset=Permission.objects.all(), required=False
    )
    business_permissions = serializers.SlugRelatedField(
        many=True, slug_field="codename", queryset=BusinessPermission.objects.all(), required=False
    )
    menus = serializers.SlugRelatedField(
        many=True, slug_field="code", queryset=Role._meta.get_field("menus").remote_field.model.objects.all(), required=False
    )

    class Meta:
        model = Role
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]


class BusinessPermissionSerializer(serializers.ModelSerializer):
    class Meta:
        model = BusinessPermission
        fields = "__all__"
        read_only_fields = ["id", "created_at"]


class PermissionSerializer(serializers.ModelSerializer):
    content_type_name = serializers.CharField(source="content_type.name", read_only=True)
    app_label = serializers.CharField(source="content_type.app_label", read_only=True)

    class Meta:
        model = Permission
        fields = ["id", "name", "codename", "content_type", "content_type_name", "app_label"]


class LoginSerializer(serializers.Serializer):
    username = serializers.CharField()
    password = serializers.CharField()
    captcha_id = serializers.CharField(required=False, allow_blank=True, default="")
    captcha_text = serializers.CharField(required=False, allow_blank=True, default="")


class LoginLogSerializer(serializers.ModelSerializer):
    class Meta:
        model = UserLoginLog
        fields = "__all__"


class BusinessCommandSerializer(serializers.ModelSerializer):
    class Meta:
        model = BusinessCommand
        fields = "__all__"
        read_only_fields = ["id", "created_at", "updated_at"]