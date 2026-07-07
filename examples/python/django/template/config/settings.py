# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

# ocimage demo settings
ALLOWED_HOSTS = ["*"]
for app in [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
]:
    INSTALLED_APPS.remove(app)
INSTALLED_APPS += ["tasks"]
for middleware in [
    "django.contrib.sessions.middleware.SessionMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
]:
    MIDDLEWARE.remove(middleware)
for context_processor in [
    "django.contrib.auth.context_processors.auth",
    "django.contrib.messages.context_processors.messages",
]:
    TEMPLATES[0]["OPTIONS"]["context_processors"].remove(context_processor)
MIDDLEWARE.insert(1, "whitenoise.middleware.WhiteNoiseMiddleware")
STATIC_ROOT = BASE_DIR / "staticfiles"
STORAGES = {
    "staticfiles": {
        "BACKEND": "whitenoise.storage.CompressedManifestStaticFilesStorage",
    },
}

DATABASES = {}
