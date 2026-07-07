<!--
Podplane <https://podplane.dev>
Copyright The Podplane Authors
SPDX-License-Identifier: Apache-2.0
-->

<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Laravel Tasks</title>
</head>
<body style="max-width: 40rem; margin: 4rem auto; font-family: system-ui, sans-serif;">
    <main>
        <h1>Laravel Tasks</h1>

        <form method="post" action="{{ route('tasks.store') }}">
            @csrf
            <input name="title" placeholder="Add a task" autofocus>
            <button type="submit">Add</button>
        </form>

        @if ($tasks)
            <ul>
                @foreach ($tasks as $task)
                    <li>{{ $task }}</li>
                @endforeach
            </ul>
        @else
            <p>No tasks yet.</p>
        @endif
    </main>
</body>
</html>
