from django.db import migrations, models


class Migration(migrations.Migration):
    dependencies = [
        ("accounts", "0009_role_is_system"),
    ]

    operations = [
        migrations.AddField(
            model_name="user",
            name="last_logout",
            field=models.DateTimeField(
                blank=True, null=True, verbose_name="最后登出时间"
            ),
        ),
    ]
