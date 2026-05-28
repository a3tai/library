import {chromium} from 'playwright';
import {mkdir} from 'node:fs/promises';
import {dirname} from 'node:path';

const url = process.env.URL || 'http://localhost:5173/?demo=1';
const out = process.argv[2] || 'screenshots/scrolled.png';
const w = Number(process.env.W || 1400);
const h = Number(process.env.H || 1100);
const dark = process.env.DARK === '1';
const scrollY = Number(process.env.SCROLL || 600);
const selector = process.env.SCROLL_SEL || '.detail-page';

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
await page.evaluate(({sel, y}) => {
  const el = document.querySelector(sel);
  if (el) el.scrollTop = y;
  else window.scrollTo(0, y);
}, {sel: selector, y: scrollY});
await page.waitForTimeout(400);
await page.screenshot({path: out, fullPage: false});
console.log(`wrote ${out} (${w}x${h}, ${dark ? 'dark' : 'light'}, scroll ${scrollY}px on ${selector})`);
await browser.close();
