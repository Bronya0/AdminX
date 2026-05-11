from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import IsAuthenticated, IsAdminUser
from rest_framework.response import Response

from .models import PasswordHistory, PasswordPolicy


@api_view(["GET"])
@permission_classes([IsAdminUser])
def password_policy(request):
    """获取当前密码策略"""
    policy = PasswordPolicy.get_instance()
    if not policy:
        return Response({"code": 200, "msg": "success", "data": {"enabled": False}})
    return Response({
        "code": 200,
        "msg": "success",
        "data": {
            "enabled": True,
            "min_length": policy.min_length,
            "require_upper": policy.require_upper,
            "require_lower": policy.require_lower,
            "require_digit": policy.require_digit,
            "require_special": policy.require_special,
            "expire_days": policy.expire_days,
            "history_count": policy.history_count,
        },
    })


@api_view(["POST"])
@permission_classes([IsAuthenticated])
def change_password(request):
    """修改密码 — 带策略校验"""
    old_password = request.data.get("old_password", "")
    new_password = request.data.get("new_password", "")

    if not request.user.check_password(old_password):
        return Response({"code": 400, "msg": "原密码不正确"})

    policy = PasswordPolicy.get_instance()
    if policy:
        valid, errors = policy.validate(new_password, user=request.user)
        if not valid:
            return Response({"code": 400, "msg": "; ".join(errors)})

    request.user.set_password(new_password)
    request.user.save(update_fields=["password"])

    if policy and policy.history_count > 0:
        PasswordHistory.record(request.user, new_password)

    return Response({"code": 200, "msg": "密码修改成功"})