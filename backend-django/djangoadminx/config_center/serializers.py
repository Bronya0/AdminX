from rest_framework import serializers

from .models import Config


class ConfigSerializer(serializers.ModelSerializer):
    display_value = serializers.SerializerMethodField()
    options = serializers.SerializerMethodField()

    class Meta:
        model = Config
        fields = [
            "id", "key", "value", "value_type", "is_encrypted",
            "desc", "group", "is_active", "display_value", "options",
            "created_at", "updated_at",
        ]
        read_only_fields = ["id", "created_at", "updated_at"]

    def get_display_value(self, obj):
        if obj.is_encrypted:
            return "********"
        if obj.value_type == Config.TypeChoices.OPTIONS:
            return obj.parse_value()
        return obj.value

    def get_options(self, obj):
        if obj.value_type == Config.TypeChoices.OPTIONS:
            return obj.parse_value()
        return None

    def to_representation(self, instance):
        data = super().to_representation(instance)
        if instance.is_encrypted:
            data["value"] = ""
        return data
