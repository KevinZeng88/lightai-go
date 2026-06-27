import { test, expect } from '@playwright/test';

test('LightAI Web can load without backend', async ({ page }) => {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'unauthorized' }),
    });
  });

  await page.goto('/');

  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('#app')).toBeVisible();

  const appText = await page.locator('#app').innerText().catch(() => '');
  expect(appText.trim().length).toBeGreaterThan(0);
});
