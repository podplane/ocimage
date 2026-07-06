// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

import { createFileRoute } from '@tanstack/solid-router'
import { createMutation, createQuery, useQueryClient } from '@tanstack/solid-query'
import { createServerFn } from '@tanstack/solid-start'
import { For, Show, createSignal } from 'solid-js'

type Task = { title: string }

const tasks: Task[] = []

const listTasks = createServerFn({ method: 'GET' }).handler(() => ({ tasks }))

const addTask = createServerFn({ method: 'POST' })
  .validator((task: Task) => task)
  .handler(({ data }) => {
    const title = data.title.trim()
    if (title) {
      tasks.push({ title })
    }
    return { result: { error: !title }, tasks }
  })

export const Route = createFileRoute('/')({
  component: Home,
})

function Home() {
  const [title, setTitle] = createSignal('')
  const queryClient = useQueryClient()
  const taskQuery = createQuery(() => ({
    queryKey: ['tasks'],
    queryFn: () => listTasks(),
    refetchInterval: 1000,
  }))
  const addTaskMutation = createMutation(() => ({
    mutationFn: addTask,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  }))
  const submitTask = () => {
    const nextTitle = title().trim()
    if (!nextTitle) return

    addTaskMutation.mutate({ data: { title: nextTitle } })
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

      <Show when={taskQuery.data?.tasks.length} fallback={<p>No tasks yet.</p>}>
        <ul>
          <For each={taskQuery.data?.tasks}>{(task) => <li>{task.title}</li>}</For>
        </ul>
      </Show>
    </main>
  )
}
