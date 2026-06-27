import { test, expect } from '@playwright/test';

test('LightAI Web can reach backend server', async ({ page, request }) => {
  const response = await request.get('http://127.0.0.1:18080/api/v1/auth/me', {
    failOnStatusCode: false,
  });

  expect(response.status(), 'backend server should be reachable').not.toBe(500);
  expect(response.status(), 'backend server should be reachable').not.toBe(502);
  expect(response.status(), 'backend server should be reachable').not.toBe(503);
  expect(response.status(), 'backend server should be reachable').not.toBe(504);

  await page.goto('/');

  await expect(page.locator('body')).toBeVisible();
  await expect(page.locator('#app')).toBeVisible();

  const appText = await page.locator('#app').innerText().catch(() => '');
  expect(appText.trim().length).toBeGreaterThan(0);
});
