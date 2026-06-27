import { defineConfig, devices } from '@playwright/test';

const baseURL = process.env.LIGHTAI_WEB_URL ?? 'http://127.0.0.1:15173';
const chromeExecutablePath = process.env.LIGHTAI_CHROME_EXECUTABLE || undefined;

export default defineConfig({
  testDir: './tests/e2e',
  outputDir: '/tmp/lightai/e2e/playwright/results',
  globalSetup: './tests/e2e/global.setup.ts',
  timeout: 60_000,
  workers: 1,
  retries: 0,
  fullyParallel: false,
  expect: {
    timeout: 10_000,
  },
  reporter: [
    ['list'],
    [
      'html',
      {
        outputFolder: '/tmp/lightai/e2e/playwright/report',
        open: 'never',
      },
    ],
  ],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
  },
  projects: [
    {
      name: 'chrome-local',
      use: {
        ...devices['Desktop Chrome'],
        browserName: 'chromium',
        launchOptions: {
          executablePath: chromeExecutablePath,
          args: [
            '--disable-gpu',
            '--disable-gpu-compositing',
            '--disable-gpu-rasterization',
            '--disable-dev-shm-usage',
            '--no-first-run',
          ],
        },
      },
    },
  ],
  webServer:
    process.env.LIGHTAI_SKIP_WEBSERVER === '1'
      ? undefined
      : {
	  command: 'npm run dev -- --host 127.0.0.1 --port 15173 --strictPort',
          url: baseURL,
          reuseExistingServer: true,
          timeout: 120_000,
          stdout: 'pipe',
          stderr: 'pipe',
        },
});
