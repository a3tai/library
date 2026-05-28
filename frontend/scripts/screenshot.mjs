import {chromium} from 'playwright';
import {mkdir} from 'node:fs/promises';
import {dirname} from 'node:path';

const url = process.env.URL || 'http://localhost:9245/';
const out = process.argv[2] || 'screenshots/app.png';
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
page.on('console', (m) => {
  if (m.type() === 'error') console.error('console.error:', m.text());
});
await page.goto(url, {waitUntil: 'networkidle', timeout: 15000});
// wait for the app to swap from "Working..." to a populated state
await page.waitForTimeout(600);
if (process.env.CLICK_NTH) {
  const idx = Number(process.env.CLICK_NTH);
  const buttons = await page.locator('.rows .row').all();
  if (buttons[idx]) await buttons[idx].click();
  await page.waitForTimeout(500);
}
await page.screenshot({path: out, fullPage: false});
console.log(`wrote ${out} (${w}x${h}, ${dark ? 'dark' : 'light'})`);
await browser.close();
