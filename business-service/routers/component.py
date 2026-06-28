"""
组件注册/心跳/注销 — 与 DjangoAdminX 平台对接

协议约定：
  POST /api/v1/cluster/components/register/    注册（幂等）
  POST /api/v1/cluster/components/heartbeat/   心跳（30s 一次），响应携带升级指令
  POST /api/v1/cluster/components/unregister/  注销

升级流程：
  1. 管理员在平台"组件管理"页面设置升级任务（填写版本号和下载地址）
  2. 业务服务心跳响应中 data.upgrade 不为 null 时触发升级
  3. 业务服务自行下载、校验、执行升级脚本
  4. 升级完成后下次心跳上报新版本，平台自动更新记录
"""

import asyncio
import hashlib
import logging
import re
import shlex
import subprocess
import tempfile
import zipfile
from pathlib import Path
from urllib.parse import urlparse

import httpx

from config import PLATFORM_URL, SERVICE_URL, APP_LABEL, APP_NAME, APP_VERSION as _INIT_VERSION

logger = logging.getLogger("business.component")

PLATFORM_API = f"{PLATFORM_URL}/api/v1/cluster/components"

# 运行时版本（升级后会更新）
APP_VERSION = _INIT_VERSION
HEARTBEAT_INTERVAL = 30  # 秒


async def register() -> bool:
    """向平台注册本服务（幂等）"""
    payload = {
        "app_label": APP_LABEL,
        "name": APP_NAME,
        "version": APP_VERSION,
        "host": SERVICE_URL,
        "description": "FastAPI 业务服务示例",
    }
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(f"{PLATFORM_API}/register/", json=payload)
            body = resp.json()
        if body.get("code") == 200:
            logger.info("组件注册成功: %s v%s", APP_LABEL, APP_VERSION)
            return True
        logger.warning("组件注册失败: %s", body.get("msg"))
        return False
    except Exception as e:
        logger.error("组件注册异常: %s", e)
        return False


async def unregister() -> bool:
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(f"{PLATFORM_API}/unregister/", json={"app_label": APP_LABEL})
            body = resp.json()
        logger.info("组件注销: %s", body.get("msg"))
        return body.get("code") == 200
    except Exception as e:
        logger.error("组件注销异常: %s", e)
        return False


async def heartbeat() -> dict | None:
    """向平台上报心跳，返回响应 data（含升级/卸载指令），失败返回 None"""
    payload = {
        "app_label": APP_LABEL,
        "version": APP_VERSION,
        "host": SERVICE_URL,
    }
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(f"{PLATFORM_API}/heartbeat/", json=payload)
            body = resp.json()
        if body.get("code") == 200:
            return body.get("data", {})
        logger.warning("心跳响应异常: %s", body.get("msg"))
        return None
    except Exception as e:
        logger.warning("心跳失败: %s", e)
        return None


def _safe_extract(zf: zipfile.ZipFile, dest: Path) -> None:
    """校验 zip 内所有条目的路径都在目标目录内，防止 zip slip"""
    dest_resolved = dest.resolve()
    for member in zf.infolist():
        member_path = (dest / member.filename).resolve()
        if not str(member_path).startswith(str(dest_resolved)):
            raise RuntimeError(f"zip slip: {member.filename} 解压路径超出目标目录")
    zf.extractall(dest)


def _validate_upgrade_url(url: str) -> None:
    """校验升级包 URL，防止 SSRF"""
    parsed = urlparse(url)
    if parsed.scheme not in ("https", "http"):
        raise ValueError(f"升级包 URL 协议不允许: {parsed.scheme}，仅支持 https/http")
    host = parsed.hostname or ""
    forbidden = ("localhost", "127.0.0.1", "0.0.0.0")
    if host in forbidden or host.startswith("192.168.") or host.startswith("10.") or host.startswith("169.254."):
        raise ValueError(f"升级包 URL 不允许内网/云元数据地址: {host}")
    # 172.16.0.0/12 范围: 172.16.x - 172.31.x
    if re.match(r"^172\.(1[6-9]|2[0-9]|3[01])\.", host):
        raise ValueError(f"升级包 URL 不允许内网地址: {host}")


async def _download_and_verify(url: str, checksum: str) -> Path:
    """下载升级包到临时目录，可选 SHA-256 校验"""
    _validate_upgrade_url(url)
    tmp = Path(tempfile.mkdtemp(prefix="upgrade_"))
    filename = url.split("/")[-1].split("?")[0] or "package"
    dest = tmp / filename
    logger.info("下载升级包: %s → %s", url, dest)
    async with httpx.AsyncClient(timeout=300, follow_redirects=True) as client:
        async with client.stream("GET", url) as resp:
            resp.raise_for_status()
            with open(dest, "wb") as f:
                async for chunk in resp.aiter_bytes(65536):
                    f.write(chunk)
    if checksum:
        actual = hashlib.sha256(dest.read_bytes()).hexdigest()
        if actual != checksum:
            raise RuntimeError(f"SHA-256 校验失败: 期望 {checksum}, 实际 {actual}")
        logger.info("包完整性校验通过")
    return dest


def _run_upgrade_script(pkg_path: Path) -> bool:
    """
    执行升级逻辑 — 根据实际部署方式修改此函数。

    示例：源码部署（解压 zip + 重启 uvicorn）
    示例：Docker 部署（docker pull + docker compose up -d）

    返回 True 表示升级成功。
    """
    suffix = pkg_path.suffix.lower()

    # ── 示例：zip 源码包 ──
    if suffix == ".zip":
        import zipfile
        work_dir = pkg_path.parent / "extracted"
        work_dir.mkdir(exist_ok=True)
        with zipfile.ZipFile(pkg_path) as zf:
            _safe_extract(zf, work_dir)
        # 执行包内的 upgrade.sh（如果存在）
        upgrade_sh = work_dir / "upgrade.sh"
        if upgrade_sh.exists():
            result = subprocess.run(
                ["bash", str(upgrade_sh)],
                cwd=str(work_dir),
                shell=False,
                capture_output=True,
                text=True,
                timeout=600,
            )
            logger.info("upgrade.sh stdout: %s", result.stdout[-2000:])
            if result.returncode != 0:
                logger.error("upgrade.sh 失败: %s", result.stderr[-2000:])
                return False
        return True

    # ── 示例：Docker 镜像包（tar）──
    if suffix in (".tar", ".gz", ".tgz"):
        result = subprocess.run(
            ["docker", "load", "-i", str(pkg_path)],
            shell=False, capture_output=True, text=True, timeout=300,
        )
        if result.returncode != 0:
            logger.error("docker load 失败: %s", result.stderr)
            return False
        # 假设 compose 文件在当前目录
        compose_file = Path(__file__).parent.parent / "docker-compose.yml"
        if compose_file.exists():
            result = subprocess.run(
                ["docker", "compose", "-f", str(compose_file), "up", "-d", "--no-build"],
                shell=False, capture_output=True, text=True, timeout=120,
            )
            if result.returncode != 0:
                logger.error("docker compose up 失败: %s", result.stderr)
                return False
        return True

    logger.warning("未知包格式 %s，跳过升级", suffix)
    return False


async def handle_upgrade(upgrade_cmd: dict) -> None:
    """处理平台下发的升级指令"""
    global APP_VERSION
    version = upgrade_cmd.get("version", "")
    url = upgrade_cmd.get("url", "")
    checksum = upgrade_cmd.get("checksum", "")

    if not url:
        logger.warning("升级指令缺少 url，跳过")
        return
    if not checksum:
        logger.error("升级指令缺少 checksum，拒绝执行（安全要求）")
        return

    logger.info("收到升级指令: v%s → v%s, url=%s", APP_VERSION, version, url)
    import shutil
    pkg_path = None
    try:
        pkg_path = await _download_and_verify(url, checksum)
        # _run_upgrade_script 内部有多个 subprocess.run（最长 600s），
        # 用 asyncio.to_thread 丢到线程池避免阻塞事件循环
        success = await asyncio.to_thread(_run_upgrade_script, pkg_path)
        if success:
            APP_VERSION = version
            logger.info("升级成功，新版本: %s", APP_VERSION)
        else:
            logger.error("升级脚本执行失败")
    except Exception as e:
        logger.error("升级失败: %s", e)
    finally:
        # 清理临时目录
        if pkg_path and pkg_path.parent.exists():
            shutil.rmtree(pkg_path.parent, ignore_errors=True)


def _run_uninstall_script() -> bool:
    """
    执行卸载逻辑 — 根据实际部署方式修改此函数。

    示例：源码部署（停止进程 + 删除目录）
    示例：Docker 部署（docker compose down + 删除镜像）

    返回 True 表示卸载成功。
    """
    # ── 示例：Docker 部署 ──
    compose_file = Path(__file__).parent.parent / "docker-compose.yml"
    if compose_file.exists():
        result = subprocess.run(
            ["docker", "compose", "-f", str(compose_file), "down", "--rmi", "local"],
            shell=False, capture_output=True, text=True, timeout=120,
        )
        if result.returncode != 0:
            logger.error("docker compose down 失败: %s", result.stderr)
            return False
        logger.info("容器已停止并移除")
        return True

    # ── 示例：源码部署（停止 systemd 服务）──
    service_name = "business-service"
    result = subprocess.run(
        ["systemctl", "stop", service_name],
        shell=False, capture_output=True, text=True, timeout=30,
    )
    if result.returncode == 0:
        logger.info("服务 %s 已停止", service_name)
        return True

    logger.warning("未找到已知的卸载方式，请手动处理")
    return False


async def handle_uninstall() -> None:
    """收到卸载指令 — 仅记录日志和标记，由管理员手动确认执行"""
    logger.warning("⚠️ 收到卸载指令！请管理员确认后手动执行卸载操作。")
    logger.warning("  可通过 docker compose down 或 systemctl stop %s 手动停止服务", APP_LABEL)


def _task_done_callback(t: asyncio.Task) -> None:
    try:
        exc = t.exception()
        if exc:
            logger.error("后台任务异常: %s", exc)
    except asyncio.CancelledError:
        pass


async def heartbeat_loop() -> None:
    """后台心跳循环 — 在 lifespan 中作为 asyncio.Task 运行"""
    logger.info("心跳任务启动，间隔 %ds", HEARTBEAT_INTERVAL)
    while True:
        data = await heartbeat()
        if data is not None:
            upgrade_cmd = data.get("upgrade")
            if upgrade_cmd:
                task = asyncio.create_task(handle_upgrade(upgrade_cmd))
                task.add_done_callback(_task_done_callback)
                await asyncio.sleep(HEARTBEAT_INTERVAL)
                continue  # 升级和卸载互斥，有升级指令时跳过卸载检查
            if data.get("uninstall"):
                task = asyncio.create_task(handle_uninstall())
                task.add_done_callback(_task_done_callback)
        await asyncio.sleep(HEARTBEAT_INTERVAL)

