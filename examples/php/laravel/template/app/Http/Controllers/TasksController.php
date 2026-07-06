<?php
// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

namespace App\Http\Controllers;

use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\View\View;

class TasksController extends Controller
{
    public function index(Request $request): View
    {
        return view('tasks', [
            'tasks' => $request->session()->get('tasks', []),
        ]);
    }

    public function store(Request $request): RedirectResponse
    {
        $title = trim((string) $request->input('title'));
        if ($title !== '') {
            $tasks = $request->session()->get('tasks', []);
            $tasks[] = $title;
            $request->session()->put('tasks', $tasks);
        }

        return redirect('/');
    }
}
