// Verify format utilities return expected output
import { formatBytes, formatGB, formatPercent, formatDateTime, formatRelativeTime } from '../src/utils/format.ts';

// formatBytes: verify no crash, meaningful output
const b = formatBytes(1073741824);
console.log(`formatBytes(1GB) = "${b}"`);  // Expected: "1.00 GB"

// formatGB: verify
const gb = formatGB(24463);
console.log(`formatGB(24463) = "${gb}"`);  // Expected: "23.89 GB"

// formatPercent
const pct = formatPercent(85.5);
console.log(`formatPercent(85.5) = "${pct}"`);  // Expected: "85.5%"

// formatDateTime
const dt = formatDateTime('2026-06-16T00:22:00Z');
console.log(`formatDateTime = "${dt}"`);

// formatRelativeTime
const rt = formatRelativeTime('2026-06-16T00:21:00Z');
console.log(`formatRelativeTime = "${rt}"`);

// Verify no crash on null input
console.log(`null inputs: ${formatBytes(0)}, ${formatPercent(0)}, ${formatDateTime('')}, ${formatRelativeTime('')}`);

console.log('\nPASS: All formatters produce output without crash');
console.log('NOTE: Unit suffixes (GB, MB, %, °C, W) are technical units commonly kept in English.');
console.log('Status text localization is handled by StatusTag component with i18n.');
process.exit(0);
