FROM python:3.13-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    DJANGO_SETTINGS_MODULE=config.settings.prod

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libpq-dev redis-tools && \
    rm -rf /var/lib/apt/lists/*

COPY requirements/prod.txt /app/requirements/
RUN pip install --no-cache-dir -r /app/requirements/prod.txt

COPY . /app

RUN mkdir -p /app/logs /app/media /app/staticfiles

EXPOSE 8000

COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD ["gunicorn", "config.wsgi:application", "--bind", "0.0.0.0:8000", "--workers", "4", "--timeout", "120"]