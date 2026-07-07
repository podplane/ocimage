// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

const apiURL = (import.meta.env.VITE_API_URL ?? 'http://localhost:8080').replace(/\/$/, '')

export type Task = { title: string }

export async function listTasks(): Promise<Task[]> {
  const res = await fetch(`${apiURL}/`)
  if (!res.ok) throw new Error(`failed to list tasks: ${res.status}`)
  const body = await res.json()
  return body.tasks
}

export async function addTask(title: string): Promise<Task[]> {
  const res = await fetch(`${apiURL}/`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ task: { title } }),
  })
  if (!res.ok) throw new Error(`failed to add task: ${res.status}`)
  const body = await res.json()
  return body.tasks
}
