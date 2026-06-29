@echo off
chcp 65001 >nul
REM Windows 开发调试脚本 — manage.py runserver 模式
REM 使用方式: 双击运行

cd /d "%~dp0"

REM 虚拟环境
set "VENV=.venv"
if not exist "%VENV%\Scripts\activate.bat" (
    echo [ERROR] 虚拟环境不存在: %VENV%
    echo     请先创建: python -m venv %VENV% ^&^& %VENV%\Scripts\activate ^&^& pip install -r requirements.txt
    pause
    exit /b 1
)
call "%VENV%\Scripts\activate.bat"

REM 默认使用 dev 配置（manage.py 已默认 config.settings.dev）
set DJANGO_SETTINGS_MODULE=config.settings.dev

echo ^>^>^> 启动调度器进程（独立窗口）...
start "DjangoAdminX Scheduler" cmd /c "cd /d %~dp0 && call .venv\Scripts\activate.bat && python manage.py run_scheduler"

echo ^>^>^> 执行数据库迁移...
python manage.py migrate --noinput

echo ^>^>^> 启动开发服务器 (127.0.0.1:9999)...
python manage.py runserver 127.0.0.1:9999

pause
