import {chromium} from 'playwright';
import {mkdir} from 'node:fs/promises';
import {dirname} from 'node:path';

const url = process.env.URL || 'http://localhost:5173/?demo=1';
const out = process.argv[2] || 'screenshots/typed.png';
const w = Number(process.env.W || 1400);
const h = Number(process.env.H || 900);
const dark = process.env.DARK === '1';
const text = process.env.TEXT || 'turing';
const clickSel = process.env.CLICK || '';

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
await page.waitForTimeout(700);
const input = page.locator('input[type="search"][aria-label="Search library"]');
await input.fill(text);
await page.waitForTimeout(700); // debounce + render
if (clickSel) {
  await page.locator(clickSel).first().click({timeout: 3000}).catch(() => {});
  await page.waitForTimeout(400);
}
await page.screenshot({path: out, fullPage: false});
console.log(`wrote ${out} (${w}x${h}, ${dark ? 'dark' : 'light'}, typed "${text}")`);
await browser.close();
