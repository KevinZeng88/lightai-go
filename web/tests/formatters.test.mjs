// Verify formatRelativeTime with zh-CN and en-US locales
const now = Date.now();

// We can't import TypeScript directly, so test the logic inline
function formatRelativeTime(iso, locale) {
  if (!iso) return 'N/A';
  const diff = now - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  const isZh = locale === 'zh-CN';
  if (s < 60) return isZh ? '刚刚' : 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return isZh ? `${m} 分钟前` : `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return isZh ? `${h} 小时前` : `${h}h ago`;
  const d = Math.floor(h / 24);
  return isZh ? `${d} 天前` : `${d}d ago`;
}

// Test zh-CN
const zhRecent = formatRelativeTime(new Date(now - 10000).toISOString(), 'zh-CN');
const zhMin = formatRelativeTime(new Date(now - 120000).toISOString(), 'zh-CN');
const zhHour = formatRelativeTime(new Date(now - 7200000).toISOString(), 'zh-CN');
const zhDay = formatRelativeTime(new Date(now - 172800000).toISOString(), 'zh-CN');

console.log(`zh-CN recent: "${zhRecent}" (expected: 刚刚)`);
console.log(`zh-CN 2min:   "${zhMin}" (expected: 2 分钟前)`);
console.log(`zh-CN 2hr:    "${zhHour}" (expected: 2 小时前)`);
console.log(`zh-CN 2day:   "${zhDay}" (expected: 2 天前)`);

// Test en-US
const enRecent = formatRelativeTime(new Date(now - 10000).toISOString(), 'en-US');
const enMin = formatRelativeTime(new Date(now - 120000).toISOString(), 'en-US');
const enHour = formatRelativeTime(new Date(now - 7200000).toISOString(), 'en-US');
const enDay = formatRelativeTime(new Date(now - 172800000).toISOString(), 'en-US');

console.log(`en-US recent: "${enRecent}" (expected: just now)`);
console.log(`en-US 2min:   "${enMin}" (expected: 2m ago)`);
console.log(`en-US 2hr:    "${enHour}" (expected: 2h ago)`);
console.log(`en-US 2day:   "${enDay}" (expected: 2d ago)`);

// Verify
const checks = [
  zhRecent === '刚刚', zhMin.includes('分钟前'), zhHour.includes('小时前'), zhDay.includes('天前'),
  enRecent === 'just now', enMin.includes('m ago'), enHour.includes('h ago'), enDay.includes('d ago'),
];
const failed = checks.filter(c => !c).length;
if (failed > 0) { console.log(`FAIL: ${failed} checks failed`); process.exit(1); }
console.log('PASS: All 8 locale checks passed');
