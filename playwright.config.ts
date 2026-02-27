import { defineConfig } from '@playwright/test'
import path from 'path'
import os from 'os'

const tmpDb = path.join(os.tmpdir(), `familycall-e2e-${Date.now()}.db`)

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  retries: 0,
  workers: 1,
  use: {
    baseURL: 'http://localhost:8089',
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  webServer: {
    command: `cd server && DATABASE_PATH=${tmpDb} TURN_PORT=13478 ./familycall --backend-only --port 8089 --frontend-uri http://localhost:8089`,
    port: 8089,
    reuseExistingServer: false,
    timeout: 10000,
  },
})
