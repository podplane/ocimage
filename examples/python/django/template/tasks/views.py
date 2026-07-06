# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

from django.shortcuts import redirect, render


def index(request):
    return render(request, "tasks/index.html", {"tasks": request.session.get("tasks", [])})


def store(request):
    title = request.POST.get("title", "").strip()
    if title:
        tasks = request.session.get("tasks", [])
        tasks.append(title)
        request.session["tasks"] = tasks
    return redirect("tasks.index")
