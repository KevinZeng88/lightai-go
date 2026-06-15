// Verify no hardcoded credentials in rendered Vue templates.
import { readFileSync, readdirSync, statSync } from 'fs';
import { join, extname } from 'path';

const SRC = 'src';
const CREDENTIAL_PATTERNS = [
  /admin\s*\/\s*lightai/i,  // default dev credentials
  /password\s*[:=]\s*['"]\S+['"]/i,  // password=value
  /admin\s*:\s*admin/i,
];

function walk(dir) {
  const results = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const st = statSync(full);
    if (st.isDirectory()) {
      results.push(...walk(full));
    } else if (extname(entry) === '.vue' || extname(entry) === '.ts') {
      results.push(full);
    }
  }
  return results;
}

// Skip i18n locale files (they define keys like 'password' as translation labels)
const SKIP_PATTERNS = [/locales\//, /__tests__\//, /\.test\./];

let found = 0;
for (const file of walk(SRC)) {
  if (SKIP_PATTERNS.some(p => p.test(file))) continue;
  const content = readFileSync(file, 'utf8');
  for (const pattern of CREDENTIAL_PATTERNS) {
    if (pattern.test(content)) {
      const lines = content.split('\n');
      for (let i = 0; i < lines.length; i++) {
        if (pattern.test(lines[i])) {
          // Allow env var references (LIGHTAI_*, GF_SECURITY_*)
          if (lines[i].includes('LIGHTAI_') || lines[i].includes('$t(') || lines[i].includes('GF_')) {
            continue;
          }
          if (lines[i].trim().startsWith('//') || lines[i].trim().startsWith('/*')) {
            continue;
          }
          console.error(`HARDCODED CREDENTIAL: ${file}:${i + 1}: ${lines[i].trim()}`);
          found++;
        }
      }
    }
  }
}

if (found > 0) {
  console.error(`FAIL: ${found} hardcoded credential(s) found`);
  process.exit(1);
} else {
  console.log('PASS: No hardcoded credentials found');
}
