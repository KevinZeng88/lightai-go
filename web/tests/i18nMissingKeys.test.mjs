// Verify all i18n keys referenced in Vue/TS templates exist in both locale files,
// AND resolve to displayable values (strings), not objects/arrays.
// Also catches non-i18n patterns like raw keys, hardcoded English/Chinese, etc.
import { readFileSync, readdirSync, statSync } from 'fs';
import { resolve, relative } from 'path';

// --- 1. Load locale keys ---
function loadLocale(path) {
  const content = readFileSync(path, 'utf-8');
  const match = content.match(/export default\s*({[\s\S]*})/);
  if (!match) throw new Error(`Cannot parse ${path}`);
  return new Function(`return ${match[1]}`)();
}

function flattenKeys(obj, prefix = '') {
  const keys = [];
  for (const [k, v] of Object.entries(obj)) {
    const full = prefix ? `${prefix}.${k}` : k;
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      keys.push(...flattenKeys(v, full));
    } else {
      keys.push(full);
    }
  }
  return keys;
}

/**
 * Build a map from full key path to the actual resolved value.
 * For leaf keys, this stores the primitive value (string/number/boolean).
 * For intermediate object keys, this stores the object itself.
 */
function buildValueMap(obj, prefix = '') {
  const map = new Map();
  for (const [k, v] of Object.entries(obj)) {
    const full = prefix ? `${prefix}.${k}` : k;
    map.set(full, v);
    if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
      const childMap = buildValueMap(v, full);
      childMap.forEach((cv, ck) => map.set(ck, cv));
    }
  }
  return map;
}

const zh = loadLocale('src/locales/zh-CN.ts');
const en = loadLocale('src/locales/en-US.ts');
const zhKeys = new Set(flattenKeys(zh));
const enKeys = new Set(flattenKeys(en));
const allKeys = new Set([...zhKeys, ...enKeys]);

// Build value maps for type-checking
const zhValueMap = buildValueMap(zh);
const enValueMap = buildValueMap(en);

/**
 * Check if a key resolves to a displayable primitive (string/number/boolean).
 * Returns { ok: boolean, reason: string }.
 *   ok=false when key resolves to an object/array (would render as [object Object]).
 *   ok=false when key doesn't exist at all (covered by missing-key check elsewhere).
 */
function checkKeyIsDisplayable(key, valueMap, keySet) {
  // Key doesn't exist at all
  if (!valueMap.has(key)) {
    // Could be a parent key like 'nav' when 'nav.backends' exists
    // That's a problem too — t('nav') returns the whole nav object
    const isParent = [...keySet].some(k => k.startsWith(key + '.'));
    if (isParent) {
      return { ok: false, reason: `resolves to object (parent of "${[...keySet].find(k => k.startsWith(key + '.'))}")` };
    }
    return { ok: false, reason: 'not found' };
  }

  const val = valueMap.get(key);
  if (typeof val === 'object' && val !== null) {
    // It's an intermediate object — t() would return the object, not a string
    const childKeys = Object.keys(val).slice(0, 3).join(', ');
    return { ok: false, reason: `resolves to object with keys: {${childKeys}${Object.keys(val).length > 3 ? '...' : ''}}` };
  }

  if (Array.isArray(val)) {
    return { ok: false, reason: 'resolves to array' };
  }

  // null/undefined shouldn't happen but check anyway
  if (val === null || val === undefined) {
    return { ok: false, reason: `resolves to ${String(val)}` };
  }

  return { ok: true, reason: 'string' };
}

// --- 2. Collect all source files ---
function collectFiles(dir, exts) {
  const files = [];
  for (const entry of readdirSync(dir)) {
    if (entry.startsWith('.') || entry === 'node_modules') continue;
    const p = resolve(dir, entry);
    if (statSync(p).isDirectory()) {
      files.push(...collectFiles(p, exts));
    } else if (exts.some(e => p.endsWith(e))) {
      files.push(p);
    }
  }
  return files;
}

const srcFiles = collectFiles('src', ['.vue', '.ts']);

// --- 3. Extract i18n keys from source ---
// Match only $t('...') or t('...') — not import(), ElMessage.error(), etc.
// The function name must be exactly 't' or '$t', preceded by a non-word char or start-of-line.
const keyPattern = /(?<!\w)(\$t|t)\s*\(\s*'([^']+)'\s*\)/g;
const allRefs = [];
const fileRefs = new Map();

for (const f of srcFiles) {
  const content = readFileSync(f, 'utf-8');
  const refs = [];
  let match;
  while ((match = keyPattern.exec(content)) !== null) {
    refs.push(match[2]);
  }
  if (refs.length > 0) {
    fileRefs.set(relative('src', f), refs);
    allRefs.push(...refs);
  }
}

// --- 4. Check each reference exists in zh-CN and en-US ---
const missingInZh = [];
const missingInEn = [];
// --- 4b. Check each reference resolves to a displayable primitive ---
const objectKeysInZh = [];
const objectKeysInEn = [];
const seen = new Set();

for (const [file, refs] of fileRefs) {
  for (const key of refs) {
    const uk = `${file}::${key}`;
    if (seen.has(uk)) continue;
    seen.add(uk);

    // Check if key is a leaf key or a parent key with children
    const keyExists = (kSet) => {
      if (kSet.has(key)) return true;
      // Check if it's a parent (e.g. 'nav' when 'nav.backends' exists)
      for (const k of kSet) {
        if (k.startsWith(key + '.')) return true;
      }
      return false;
    };

    if (!keyExists(zhKeys)) {
      missingInZh.push(`${file}: ${key}`);
    }
    if (!keyExists(enKeys)) {
      missingInEn.push(`${file}: ${key}`);
    }

    // NEW: Check that the key resolves to a displayable primitive, not an object
    const zhDisplayable = checkKeyIsDisplayable(key, zhValueMap, zhKeys);
    const enDisplayable = checkKeyIsDisplayable(key, enValueMap, enKeys);

    if (!zhDisplayable.ok) {
      objectKeysInZh.push(`${file}: ${key} → ${zhDisplayable.reason}`);
    }
    if (!enDisplayable.ok) {
      objectKeysInEn.push(`${file}: ${key} → ${enDisplayable.reason}`);
    }
  }
}

// --- 5. Scan for hardcoded i18n-key-like patterns in templates (key leaks) ---
// Pattern: looks like an i18n key (word.word) displayed directly without t()
const hardcodedIssues = [];
for (const f of srcFiles) {
  const content = readFileSync(f, 'utf-8');
  // Look for Vue template sections with bare dot-separated identifiers
  // inside text nodes that look like i18n keys
  const templateMatch = content.match(/<template>([\s\S]*)<\/template>/);
  if (!templateMatch) continue;
  const template = templateMatch[1];

  // Find text between > and < that looks like an i18n key (e.g. "nav.modelArtifacts")
  const textPattern = />([^<]{2,40})</g;
  let tm;
  while ((tm = textPattern.exec(template)) !== null) {
    const text = tm[1].trim();
    // Check if it looks like an i18n key: word.word.word pattern
    if (/^[a-z]+\.[a-zA-Z]+\.[a-zA-Z]+$/.test(text) || /^[a-z]+\.[a-zA-Z]+$/.test(text)) {
      // Exclude known non-i18n patterns
      if (text.startsWith('el-') || text.startsWith('v-') || text.startsWith('$')) continue;
      if (/^(true|false|null|undefined)$/.test(text)) continue;
      hardcodedIssues.push(`${relative('src', f)}: raw key-like text "${text}"`);
    }
  }
}

// --- 6. Report ---
let exitCode = 0;

if (missingInZh.length > 0) {
  console.log(`\nMISSING from zh-CN (${missingInZh.length}):`);
  missingInZh.forEach(k => console.log(`  ${k}`));
  exitCode = 1;
}
if (missingInEn.length > 0) {
  console.log(`\nMISSING from en-US (${missingInEn.length}):`);
  missingInEn.forEach(k => console.log(`  ${k}`));
  exitCode = 1;
}

// NEW: Report keys that resolve to objects instead of displayable strings
if (objectKeysInZh.length > 0) {
  console.log(`\nOBJECT-VALUE keys in zh-CN (${objectKeysInZh.length}) — t() would render [object Object]:`);
  objectKeysInZh.forEach(k => console.log(`  ${k}`));
  exitCode = 1;
}
if (objectKeysInEn.length > 0) {
  console.log(`\nOBJECT-VALUE keys in en-US (${objectKeysInEn.length}) — t() would render [object Object]:`);
  objectKeysInEn.forEach(k => console.log(`  ${k}`));
  exitCode = 1;
}

if (hardcodedIssues.length > 0) {
  console.log(`\nPOTENTIAL HARDCODED i18n key leaks (${hardcodedIssues.length}):`);
  hardcodedIssues.forEach(k => console.log(`  ${k}`));
}

if (exitCode === 0) {
  console.log(`PASS: all ${seen.size} i18n key references found in both locale files and resolve to strings`);
  console.log(`  zh-CN leaf keys: ${zhKeys.size}`);
  console.log(`  en-US leaf keys: ${enKeys.size}`);
}
if (hardcodedIssues.length > 0) {
  console.log(`\nWARNING: ${hardcodedIssues.length} potential hardcoded i18n key leaks found (review manually)`);
  // Don't fail on warnings — some may be false positives
}

process.exit(exitCode);
