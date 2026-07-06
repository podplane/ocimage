// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

import { tanstackStart } from '@tanstack/solid-start/plugin/vite'
import { defineConfig } from 'vite'
import { nitro } from 'nitro/vite'
import viteSolid from 'vite-plugin-solid'

const runtime = process.env.RUNTIME === 'bun' ? 'bun' : 'node-server'

export default defineConfig({
  server: {
    port: 8080,
  },
  plugins: [tanstackStart(), nitro({ preset: runtime }), viteSolid({ ssr: true })],
})
