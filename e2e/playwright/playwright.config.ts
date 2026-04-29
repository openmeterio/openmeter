import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: process.env.OPENMETER_ADDRESS ?? 'http://localhost:8888',
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...(process.env.OPENMETER_API_KEY
        ? { Authorization: `Bearer ${process.env.OPENMETER_API_KEY}` }
        : {}),
    },
    ignoreHTTPSErrors: true,
  },
})
