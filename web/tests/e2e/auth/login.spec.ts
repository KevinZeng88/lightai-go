import { test, expect } from '@playwright/test';
import { adminStorageStatePath } from '../helpers/auth';

test.use({ storageState: adminStorageStatePath });

test('admin storage state can access authenticated app', async ({ page }) => {
  await page.goto('/');

  const me = await page.evaluate(async () => {
    const response = await fetch('/api/v1/auth/me', {
      credentials: 'include',
    });

    return {
      status: response.status,
      text: await response.text(),
    };
  });

  expect(me.status, me.text).toBe(200);

  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('#app')).toBeVisible();

  const appText = await page.locator('#app').innerText().catch(() => '');
  expect(appText.trim().length).toBeGreaterThan(0);

  await expect(page.getByText(/Administrator @ Default Tenant/)).toBeVisible();
  await expect(page.getByText(/^登录$/)).toHaveCount(0);
});
