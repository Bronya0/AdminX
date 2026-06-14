from django.core.cache import cache
from rest_framework.decorators import api_view, permission_classes
from rest_framework.permissions import IsAdminUser
from rest_framework.response import Response


@api_view(["GET"])
@permission_classes([IsAdminUser])
def cache_stats(request):
    """缓存统计"""
    try:
        # 尝试获取 Redis 信息
        client = cache.client.get_client() if hasattr(cache, "client") else None
        if client and hasattr(client, "info"):
            info = client.info()
            data = {
                "backend": "redis",
                "used_memory": info.get("used_memory_human", "N/A"),
                "keys": info.get("db0", {}).get("keys", "N/A") if "db0" in info else "N/A",
                "uptime_days": info.get("uptime_in_days", "N/A"),
                "hit_rate": f"{info.get('keyspace_hitrate', 'N/A')}" if "keyspace_hitrate" in info else "N/A",
            }
        else:
            data = {"backend": "locmem", "msg": "本地内存缓存，不支持统计"}
    except Exception as e:
        data = {"backend": "unknown", "msg": str(e)[:200]}

    return Response({"code": 200, "msg": "success", "data": data})


@api_view(["POST"])
@permission_classes([IsAdminUser])
def cache_clear(request):
    """清理缓存 — 带 prefix 参数可选"""
    prefix = request.data.get("prefix", "")

    try:
        if prefix:
            # 按前缀清理 (需要 Redis)
            client = cache.client.get_client() if hasattr(cache, "client") else None
            if client and hasattr(client, "keys"):
                keys = client.keys(f"{prefix}*")
                if keys:
                    client.delete(*keys)
                count = len(keys)
            else:
                cache.clear()
                count = "all (locmem)"
        else:
            cache.clear()
            count = "all"
    except Exception as e:
        return Response({"code": 500, "msg": str(e)[:200]})

    return Response({"code": 200, "msg": "success", "data": {"cleared": count}})