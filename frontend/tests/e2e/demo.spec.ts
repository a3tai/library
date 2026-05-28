import {expect, test} from '@playwright/test';

test('loads the demo library without the Wails runtime', async ({page}) => {
  await page.goto('/?demo');

  await expect(page.getByText('The Annotated Turing')).toBeVisible();
  await expect(page.getByText('A Pattern Language')).toBeVisible();
  await expect(page.getByText('Charles Petzold')).toBeVisible();
});
