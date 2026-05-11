"""WebService 任务处理函数 — 被 APScheduler 动态导入调用"""

import json
import logging

logger = logging.getLogger("djangoadminx.webservice.tasks")


def call_webservice(webservice_id, params=None):
    """调用外部 WebService (通用任务)"""
    from djangoadminx.webservice.models import WebService
    from zeep import Client

    ws = WebService.objects.get(id=webservice_id)
    client = Client(ws.wsdl_url)
    method = getattr(client.service, ws.method)
    params = params or {}
    result = method(**params)
    logger.info(f"WebService call {ws.name}.{ws.method}: {result}")
    return str(result)


def ntp_sync(server="pool.ntp.org", version=3):
    """NTP 时间同步任务"""
    import ntplib
    client = ntplib.NTPClient()
    response = client.request(server, version=version)
    return json.dumps({
        "server": server,
        "offset": response.offset,
        "delay": response.delay,
        "time": str(response.tx_time),
    })


def sample_task():
    """示例任务"""
    logger.info("Sample task executed")
    return "ok"