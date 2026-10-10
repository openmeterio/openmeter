import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true, // Use Vitest's globals (describe, it, expect, vi) without importing
    environment: 'node', // Set the test environment to Node.js
    // Automatically clear mock history between tests
    clearMocks: true,
    // You can specify include patterns if your tests aren't named *.test.ts or *.spec.ts
    // include: ['src/**/*.test.ts'],
  },
})
