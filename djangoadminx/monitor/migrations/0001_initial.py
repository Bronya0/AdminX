from django.db import migrations, models
import django.utils.timezone


class Migration(migrations.Migration):

    initial = True

    dependencies = []

    operations = [
        migrations.CreateModel(
            name="SystemMetricSnapshot",
            fields=[
                ("id", models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name="ID")),
                ("collected_at", models.DateTimeField(db_index=True, default=django.utils.timezone.now, verbose_name="采集时间")),
                ("cpu_percent", models.FloatField(default=0, verbose_name="CPU 使用率")),
                ("memory_percent", models.FloatField(default=0, verbose_name="内存使用率")),
                ("disk_percent", models.FloatField(default=0, verbose_name="磁盘使用率")),
                ("disk_read_bytes", models.BigIntegerField(default=0, verbose_name="磁盘累计读取字节")),
                ("disk_write_bytes", models.BigIntegerField(default=0, verbose_name="磁盘累计写入字节")),
            ],
            options={
                "verbose_name": "系统资源快照",
                "verbose_name_plural": "系统资源快照",
                "ordering": ["-collected_at"],
            },
        ),
    ]
