# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

from django.urls import path

from tasks import views

urlpatterns = [
    path("", views.index, name="tasks.index"),
    path("tasks", views.store, name="tasks.store"),
]
