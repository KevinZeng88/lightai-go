// Verify zh-CN and en-US i18n keys have matching structure
import { readFileSync } from 'fs';

function extractKeys(obj, prefix = '') {
  const keys = [];
  for (const [k, v] of Object.entries(obj)) {
    const full = prefix ? `${prefix}.${k}` : k;
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      keys.push(...extractKeys(v, full));
    } else {
      keys.push(full);
    }
  }
  return keys.sort();
}

function loadLocale(path) {
  const content = readFileSync(path, 'utf-8');
  // Extract the default export object
  const match = content.match(/export default\s*({[\s\S]*})/);
  if (!match) throw new Error(`Cannot parse ${path}`);
  // Use Function constructor to safely evaluate
  const obj = new Function(`return ${match[1]}`)();
  return obj;
}

const zh = loadLocale('src/locales/zh-CN.ts');
const en = loadLocale('src/locales/en-US.ts');

const zhKeys = new Set(extractKeys(zh));
const enKeys = new Set(extractKeys(en));

const onlyZh = [...zhKeys].filter(k => !enKeys.has(k));
const onlyEn = [...enKeys].filter(k => !zhKeys.has(k));

console.log(`zh-CN keys: ${zhKeys.size}`);
console.log(`en-US keys: ${enKeys.size}`);

if (onlyZh.length) {
  console.log(`Keys only in zh-CN (${onlyZh.length}):`);
  onlyZh.slice(0, 20).forEach(k => console.log(`  ${k}`));
}
if (onlyEn.length) {
  console.log(`Keys only in en-US (${onlyEn.length}):`);
  onlyEn.slice(0, 20).forEach(k => console.log(`  ${k}`));
}

if (!onlyZh.length && !onlyEn.length) {
  console.log('PASS: i18n keys consistent between zh-CN and en-US');
  process.exit(0);
} else {
  console.log('FAIL: i18n key mismatch');
  process.exit(1);
}
