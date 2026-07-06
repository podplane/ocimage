// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

import { Hono } from 'hono'
import { cors } from 'hono/cors'

const app = new Hono()
type Task = { title: string }

const tasks: Task[] = []

app.use('*', cors())

app.get('/', (c) => c.json({ tasks }))

app.post('/', async (c) => {
  const contentType = c.req.header('content-type') ?? ''
  let title = ''
  if (contentType.includes('application/json')) {
    const body = await c.req.json<{ task?: { title?: string } }>().catch(() => ({}))
    title = String(body.task?.title ?? '').trim()
  } else {
    title = (await c.req.text()).trim()
  }
  if (!title) {
    return c.json({ result: { error: true, message: 'Task title is required.' } }, 400)
  }
  tasks.push({ title })
  return c.json({ result: { error: false }, tasks }, 201)
})

const port = Number((typeof Bun === 'undefined' ? process.env.PORT : Bun.env.PORT) ?? 8080)

if (typeof Bun !== 'undefined') {
  Bun.serve({ port, fetch: app.fetch })
} else {
  const { serve } = await import('@hono/node-server')
  serve({ port, fetch: app.fetch })
}

console.log(`listening on http://localhost:${port}`)
