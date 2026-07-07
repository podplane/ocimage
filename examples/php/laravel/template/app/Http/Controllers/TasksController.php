<?php
// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

namespace App\Http\Controllers;

use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\View\View;

class TasksController extends Controller
{
    private const TasksKey = 'ocimage-laravel-tasks';

    public function index(): View
    {
        return view('tasks', [
            'tasks' => Cache::store('file')->get(self::TasksKey, []),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        $title = trim((string) $request->input('title'));
        if ($title !== '') {
            $tasks = Cache::store('file')->get(self::TasksKey, []);
            $tasks[] = $title;
            Cache::store('file')->forever(self::TasksKey, $tasks);
        }

        return redirect('/');
    }
}
