import { chromium, type FullConfig } from '@playwright/test';
import { adminStorageStatePath, ensureAdminLoggedIn } from './helpers/auth';

async function globalSetup(config: FullConfig): Promise<void> {
  if (process.env.LIGHTAI_SKIP_AUTH_SETUP === '1') {
    return;
  }

  const project = config.projects.find((item) => item.name === 'chrome-local') ?? config.projects[0];
  const baseURL = process.env.LIGHTAI_WEB_URL ?? 'http://127.0.0.1:15173';
  const executablePath = process.env.LIGHTAI_CHROME_EXECUTABLE || undefined;

  const browser = await chromium.launch({
    headless: true,
    executablePath,
    args: [
      '--disable-gpu',
      '--disable-gpu-compositing',
      '--disable-gpu-rasterization',
      '--disable-dev-shm-usage',
      '--no-first-run',
    ],
  });

  const context = await browser.newContext({
    baseURL,
    ...(project.use.viewport ? { viewport: project.use.viewport } : {}),
  });

  const page = await context.newPage();

  try {
    await ensureAdminLoggedIn(page);
    await context.storageState({ path: adminStorageStatePath });
  } finally {
    await browser.close();
  }
}

export default globalSetup;
