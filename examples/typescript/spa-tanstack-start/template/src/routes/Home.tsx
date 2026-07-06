// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

import { createMutation, createQuery, useQueryClient } from '@tanstack/solid-query'
import { For, Show, createSignal } from 'solid-js'

import { addTask, listTasks } from '../api/tasks'

export function Home() {
  const [title, setTitle] = createSignal('')
  const client = useQueryClient()
  const taskQuery = createQuery(() => ({
    queryKey: ['tasks'],
    queryFn: listTasks,
    refetchInterval: 1000,
  }))
  const addTaskMutation = createMutation(() => ({
    mutationFn: addTask,
    onSuccess: () => client.invalidateQueries({ queryKey: ['tasks'] }),
  }))
  const submitTask = () => {
    const nextTitle = title().trim()
    if (!nextTitle) return

    addTaskMutation.mutate(nextTitle)
    setTitle('')
  }

  return (
    <main style="max-width: 40rem; margin: 4rem auto; font-family: system-ui, sans-serif;">
      <h1>Tasks</h1>

      <form
        onSubmit={(event) => {
          event.preventDefault()
          submitTask()
        }}
      >
        <input
          value={title()}
          onInput={(event) => setTitle(event.currentTarget.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault()
              submitTask()
            }
          }}
          placeholder="Add a task"
          autofocus
        />{' '}
        <button type="submit" disabled={addTaskMutation.isPending}>
          Add
        </button>
      </form>

      <Show when={taskQuery.data?.length} fallback={<p>No tasks yet.</p>}>
        <ul>
          <For each={taskQuery.data}>{(task) => <li>{task.title}</li>}</For>
        </ul>
      </Show>
    </main>
  )
}
