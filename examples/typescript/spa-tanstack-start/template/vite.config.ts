// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

import { defineConfig } from 'vite'
import viteSolid from 'vite-plugin-solid'

export default defineConfig({
  server: {
    port: 8080,
  },
  plugins: [viteSolid()],
})
