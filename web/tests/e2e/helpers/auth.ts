import { expect, type Locator, type Page } from '@playwright/test';

export const adminStorageStatePath = 'tests/e2e/.auth/admin.json';

export type AdminCredentials = {
  username: string;
  password: string;
  newPassword: string;
};

export function getAdminCredentials(): AdminCredentials {
  const username = process.env.LIGHTAI_E2E_ADMIN_USERNAME;
  const password = process.env.LIGHTAI_E2E_ADMIN_PASSWORD;
  const newPassword = process.env.LIGHTAI_E2E_ADMIN_NEW_PASSWORD ?? 'LightAI-Test-Admin-2026!';

  if (!username || !password) {
    throw new Error(
      [
        'Missing E2E admin credentials.',
        'Set these environment variables before running Playwright:',
        '  LIGHTAI_E2E_ADMIN_USERNAME',
        '  LIGHTAI_E2E_ADMIN_PASSWORD',
        'Optional:',
        '  LIGHTAI_E2E_ADMIN_NEW_PASSWORD',
      ].join('\n'),
    );
  }

  return { username, password, newPassword };
}

async function firstVisible(locators: Locator[], timeoutMs = 2500): Promise<Locator | null> {
  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    for (const locator of locators) {
      if (await locator.first().isVisible().catch(() => false)) {
        return locator.first();
      }
    }

    await new Promise((resolve) => setTimeout(resolve, 100));
  }

  return null;
}

async function fillFirstVisible(locators: Locator[], value: string, fieldName: string): Promise<void> {
  const locator = await firstVisible(locators);

  if (!locator) {
    throw new Error(`Cannot find visible ${fieldName} field`);
  }

  await locator.fill(value);
}

async function clickFirstVisible(locators: Locator[], actionName: string): Promise<void> {
  const locator = await firstVisible(locators);

  if (!locator) {
    throw new Error(`Cannot find visible ${actionName}`);
  }

  await locator.click();
}

export async function isAuthenticated(page: Page): Promise<boolean> {
  const status = await page.evaluate(async () => {
    const response = await fetch('/api/v1/auth/me', {
      credentials: 'include',
    });

    return response.status;
  });

  return status === 200;
}

export async function loginWithPassword(page: Page, username: string, password: string): Promise<void> {
  await page.goto('/');

  if (await isAuthenticated(page)) {
    return;
  }

  await fillFirstVisible(
    [
      page.getByLabel(/用户名|账号|用户|邮箱|email|username|account/i),
      page.locator('input[name="username"]'),
      page.locator('input[name="email"]'),
      page.locator('input[type="email"]'),
      page.locator('input[type="text"]').first(),
    ],
    username,
    'username',
  );

  await fillFirstVisible(
    [
      page.getByLabel(/密码|password/i),
      page.locator('input[name="password"]'),
      page.locator('input[type="password"]').first(),
    ],
    password,
    'password',
  );

  await clickFirstVisible(
    [
      page.getByRole('button', { name: /登录|登陆|sign in|log in|login/i }),
      page.locator('button[type="submit"]'),
    ],
    'login button',
  );

  await page.waitForLoadState('networkidle').catch(() => undefined);
}

export async function maybeChangeInitialPassword(
  page: Page,
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const changePasswordSignal = await firstVisible(
    [
      page.getByText(/修改密码|更改密码|首次登录|change password|reset password/i),
      page.getByLabel(/新密码|new password/i),
      page.locator('input[name="new_password"]'),
      page.locator('input[name="newPassword"]'),
    ],
    3000,
  );

  if (!changePasswordSignal) {
    return;
  }

  const passwordInputs = page.locator('input[type="password"]');
  const count = await passwordInputs.count();

  if (count >= 3) {
    await passwordInputs.nth(0).fill(currentPassword);
    await passwordInputs.nth(1).fill(newPassword);
    await passwordInputs.nth(2).fill(newPassword);
  } else if (count >= 2) {
    await passwordInputs.nth(0).fill(newPassword);
    await passwordInputs.nth(1).fill(newPassword);
  } else {
    throw new Error('Change password page detected but password fields are not recognizable');
  }

  await clickFirstVisible(
    [
      page.getByRole('button', { name: /保存|提交|确认|修改|change|save|submit|confirm/i }),
      page.locator('button[type="submit"]'),
    ],
    'change password submit button',
  );

  await page.waitForLoadState('networkidle').catch(() => undefined);
}

export async function ensureAdminLoggedIn(page: Page): Promise<void> {
  const { username, password, newPassword } = getAdminCredentials();

  await loginWithPassword(page, username, password);
  await maybeChangeInitialPassword(page, password, newPassword);

  if (!(await isAuthenticated(page))) {
    await loginWithPassword(page, username, newPassword);
  }

  expect(await isAuthenticated(page), 'admin should be authenticated after login setup').toBe(true);
}
