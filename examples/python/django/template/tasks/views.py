# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

from threading import Lock

from django.shortcuts import redirect, render


_tasks = []
_tasks_lock = Lock()


def index(request):
    with _tasks_lock:
        tasks = list(_tasks)
    return render(request, "tasks/index.html", {"tasks": tasks})


def store(request):
    title = request.POST.get("title", "").strip()
    if title:
        with _tasks_lock:
            _tasks.append(title)
    return redirect("tasks.index")
