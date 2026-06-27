import { test, expect } from '@playwright/test';
import { getAdminCredentials, loginWithPassword, maybeChangeInitialPassword } from '../helpers/auth';

test('debug admin login flow', async ({ page }) => {
  const { username, password, newPassword } = getAdminCredentials();

  const responses: string[] = [];

  page.on('response', async (response) => {
    const url = response.url();

    if (!url.includes('/api/')) {
      return;
    }

    let body = '';

    try {
      body = await response.text();
    } catch {
      body = '<unreadable>';
    }

    responses.push(`${response.status()} ${url}\n${body.slice(0, 500)}`);
  });

  await loginWithPassword(page, username, password);

  console.log('After login URL:', page.url());
  console.log('After login title:', await page.title().catch(() => ''));

  await page.screenshot({
    path: '/tmp/lightai/e2e/playwright/login-after-login.png',
    fullPage: true,
  });

  console.log(
    'After login page text:',
    (await page.locator('body').innerText().catch(() => '')).slice(0, 2000),
  );

  const storageAfterLogin = await page.evaluate(() => {
    return {
      localStorage: Object.fromEntries(Object.entries(localStorage)),
      sessionStorage: Object.fromEntries(Object.entries(sessionStorage)),
      cookiesEnabled: navigator.cookieEnabled,
    };
  });

  console.log('Storage after login:', JSON.stringify(storageAfterLogin, null, 2));

  await maybeChangeInitialPassword(page, password, newPassword);

  console.log('After maybeChangeInitialPassword URL:', page.url());

  await loginWithPassword(page, username, newPassword);

  console.log('After relogin with new password URL:', page.url());

  await page.screenshot({
    path: '/tmp/lightai/e2e/playwright/login-after-change-password.png',
    fullPage: true,
  });

  console.log(
    'After maybeChangeInitialPassword page text:',
    (await page.locator('body').innerText().catch(() => '')).slice(0, 2000),
  );

  const authMeViaBrowser = await page.evaluate(async () => {
    const response = await fetch('/api/v1/auth/me', {
      credentials: 'include',
    });

    return {
      status: response.status,
      text: await response.text(),
    };
  });

  console.log('auth/me via browser fetch:', JSON.stringify(authMeViaBrowser, null, 2));

  const storageAfterChange = await page.evaluate(() => {
    return {
      localStorage: Object.fromEntries(Object.entries(localStorage)),
      sessionStorage: Object.fromEntries(Object.entries(sessionStorage)),
      cookiesEnabled: navigator.cookieEnabled,
    };
  });

  console.log('Storage after maybeChangeInitialPassword:', JSON.stringify(storageAfterChange, null, 2));

  console.log('API responses:\n' + responses.join('\n\n---\n\n'));

  expect(page.locator('body')).toBeVisible();
});
