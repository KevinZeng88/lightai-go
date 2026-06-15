// Verify all API client modules use relative paths
import { readFileSync, readdirSync } from 'fs';

const apiDir = 'src/api';
const files = readdirSync(apiDir).filter(f => f.endsWith('.ts') && f !== 'client.ts');

let passed = 0;
let failed = 0;

for (const f of files) {
  const content = readFileSync(`${apiDir}/${f}`, 'utf-8');
  // Check for hardcoded /api/v1 prefix in apiClient calls
  const hardcodedPattern = /apiClient\.\w+\(['"]\/api\/v1\//g;
  const matches = content.match(hardcodedPattern);
  if (matches) {
    console.log(`FAIL: ${f} has hardcoded /api/v1 prefix (${matches.length} occurrences)`);
    failed++;
  } else {
    passed++;
  }
}

console.log(`\nPassed: ${passed}, Failed: ${failed}`);
if (failed > 0) {
  console.log('FAIL: Some modules have hardcoded /api/v1 prefix');
  process.exit(1);
} else {
  console.log('PASS: All API modules use relative paths');
  process.exit(0);
}
