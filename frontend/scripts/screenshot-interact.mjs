import {chromium} from 'playwright';
import {mkdir} from 'node:fs/promises';
import {dirname} from 'node:path';

const url = process.env.URL || 'http://localhost:9245/';
const out = process.argv[2] || 'screenshots/app.png';
const action = process.env.ACTION || ''; // 'toc' | 'settings' | 'lightbox'
const w = Number(process.env.W || 1280);
const h = Number(process.env.H || 820);
const dark = process.env.DARK === '1';

await mkdir(dirname(out), {recursive: true});
const browser = await chromium.launch();
const ctx = await browser.newContext({
  viewport: {width: w, height: h},
  deviceScaleFactor: 2,
  colorScheme: dark ? 'dark' : 'light',
});
const page = await ctx.newPage();
page.on('pageerror', (e) => console.error('pageerror:', e.message));
await page.goto(url, {waitUntil: 'networkidle', timeout: 15000});
await page.waitForTimeout(500);

if (action === 'toc') {
  await page.locator('button[title="Contents (T)"]').click();
  await page.waitForTimeout(200);
} else if (action === 'settings') {
  await page.locator('button[title="Display settings"]').click();
  await page.waitForTimeout(200);
} else if (action === 'lightbox') {
  await page.locator('button.cover-button').click();
  await page.waitForTimeout(200);
}

await page.screenshot({path: out, fullPage: false});
console.log(`wrote ${out} (${w}x${h}, ${dark ? 'dark' : 'light'}, action=${action || 'none'})`);
await browser.close();
