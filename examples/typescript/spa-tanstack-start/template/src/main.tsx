// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { RouterProvider } from '@tanstack/solid-router'
import { render } from 'solid-js/web'

import { router } from './router'

const queryClient = new QueryClient()

render(
  () => (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  ),
  document.getElementById('root')!,
)
