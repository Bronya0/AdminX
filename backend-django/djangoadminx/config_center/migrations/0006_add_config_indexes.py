from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ("config_center", "0005_alter_config_value_type"),
    ]

    operations = [
        migrations.AddIndex(
            model_name="config",
            index=models.Index(
                fields=["group", "is_active", "key"],
                name="config_group_active_key_idx",
            ),
        ),
        migrations.AddIndex(
            model_name="config",
            index=models.Index(
                fields=["is_active", "value_type", "created_at"],
                name="config_active_type_created_idx",
            ),
        ),
        migrations.AddIndex(
            model_name="config",
            index=models.Index(
                fields=["is_active", "is_encrypted", "created_at"],
                name="config_act_enc_created_idx",
            ),
        ),
    ]
