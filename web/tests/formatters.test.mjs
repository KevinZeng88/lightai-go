// Verify formatRelativeTime with zh-CN and en-US locales
const now = Date.now();

// Inline implementation matching web/src/utils/format.ts
function formatRelativeTime(iso, locale) {
  if (!iso) return '-';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '-';

  const diff = now - d.getTime();
  const s = Math.floor(diff / 1000);
  const isZh = locale === 'zh-CN';

  // Future time (slight clock skew — treat as just now).
  if (s < 0) {
    if (s > -5) return isZh ? '<1秒前' : '<1s ago';
    return isZh ? '时间异常' : 'time anomaly';
  }

  // < 1 second
  if (s < 1) return isZh ? '<1秒前' : '<1s ago';

  // 1-59 seconds
  if (s < 60) return isZh ? `${s}秒前` : `${s}s ago`;

  // 1-59 minutes
  const m = Math.floor(s / 60);
  if (m < 60) return isZh ? `${m}分钟前` : `${m}m ago`;

  // 1-24 hours
  const h = Math.floor(m / 60);
  if (h < 24) return isZh ? `${h}小时前` : `${h}h ago`;

  // > 24 hours: show absolute date/time.
  const pad = (n) => String(n).padStart(2, '0');
  const nowDate = new Date();

  // Different year → YYYY-MM-DD HH:mm:ss
  if (d.getFullYear() !== nowDate.getFullYear()) {
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
  }

  // Same year → MM-DD HH:mm:ss
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

// ---- Test cases ----
let failures = 0;

function check(label, actual, expectedFn) {
  const ok = expectedFn(actual);
  if (!ok) {
    console.log(`FAIL [${label}]: got "${actual}"`);
    failures++;
  } else {
    console.log(`PASS [${label}]: "${actual}"`);
  }
}

// zh-CN tests
check('zh-CN null', formatRelativeTime(null, 'zh-CN'), v => v === '-');
check('zh-CN empty', formatRelativeTime('', 'zh-CN'), v => v === '-');
check('zh-CN invalid', formatRelativeTime('not-a-date', 'zh-CN'), v => v === '-');
check('zh-CN <1s', formatRelativeTime(new Date(now - 500).toISOString(), 'zh-CN'), v => v === '<1秒前');
check('zh-CN 2s', formatRelativeTime(new Date(now - 2000).toISOString(), 'zh-CN'), v => v === '2秒前');
check('zh-CN 30s', formatRelativeTime(new Date(now - 30000).toISOString(), 'zh-CN'), v => v === '30秒前');
check('zh-CN 59s', formatRelativeTime(new Date(now - 59000).toISOString(), 'zh-CN'), v => v === '59秒前');
check('zh-CN 1min', formatRelativeTime(new Date(now - 60000).toISOString(), 'zh-CN'), v => v === '1分钟前');
check('zh-CN 5min', formatRelativeTime(new Date(now - 300000).toISOString(), 'zh-CN'), v => v === '5分钟前');
check('zh-CN 59min', formatRelativeTime(new Date(now - 3540000).toISOString(), 'zh-CN'), v => v === '59分钟前');
check('zh-CN 1hr', formatRelativeTime(new Date(now - 3600000).toISOString(), 'zh-CN'), v => v === '1小时前');
check('zh-CN 23hr', formatRelativeTime(new Date(now - 82800000).toISOString(), 'zh-CN'), v => v === '23小时前');
check('zh-CN 25hr (same year)', formatRelativeTime(new Date(now - 90000000).toISOString(), 'zh-CN'), v => /^\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(v));

// Future time tests
check('zh-CN future 2s', formatRelativeTime(new Date(now + 2000).toISOString(), 'zh-CN'), v => v === '<1秒前');
check('zh-CN future 10s', formatRelativeTime(new Date(now + 10000).toISOString(), 'zh-CN'), v => v === '时间异常');

// en-US tests
check('en-US null', formatRelativeTime(null, 'en-US'), v => v === '-');
check('en-US invalid', formatRelativeTime('bad', 'en-US'), v => v === '-');
check('en-US <1s', formatRelativeTime(new Date(now - 500).toISOString(), 'en-US'), v => v === '<1s ago');
check('en-US 2s', formatRelativeTime(new Date(now - 2000).toISOString(), 'en-US'), v => v === '2s ago');
check('en-US 59s', formatRelativeTime(new Date(now - 59000).toISOString(), 'en-US'), v => v === '59s ago');
check('en-US 1min', formatRelativeTime(new Date(now - 60000).toISOString(), 'en-US'), v => v === '1m ago');
check('en-US 59min', formatRelativeTime(new Date(now - 3540000).toISOString(), 'en-US'), v => v === '59m ago');
check('en-US 1hr', formatRelativeTime(new Date(now - 3600000).toISOString(), 'en-US'), v => v === '1h ago');
check('en-US 23hr', formatRelativeTime(new Date(now - 82800000).toISOString(), 'en-US'), v => v === '23h ago');
check('en-US future', formatRelativeTime(new Date(now + 2000).toISOString(), 'en-US'), v => v === '<1s ago');

// Verify no "刚刚" or "just now" anywhere
const allOutputs = [
  formatRelativeTime(new Date(now - 500).toISOString(), 'zh-CN'),
  formatRelativeTime(new Date(now - 2000).toISOString(), 'zh-CN'),
  formatRelativeTime(new Date(now - 15000).toISOString(), 'zh-CN'),
  formatRelativeTime(new Date(now - 500).toISOString(), 'en-US'),
  formatRelativeTime(new Date(now - 2000).toISOString(), 'en-US'),
  formatRelativeTime(new Date(now - 15000).toISOString(), 'en-US'),
];
if (allOutputs.some(v => v === '刚刚' || v === 'just now')) {
  console.log('FAIL: "刚刚" or "just now" found in output');
  failures++;
} else {
  console.log('PASS: No "刚刚" or "just now" in any output');
}

// Cross-year test
const lastYear = new Date(now);
lastYear.setFullYear(lastYear.getFullYear() - 1);
const crossYearResult = formatRelativeTime(lastYear.toISOString(), 'zh-CN');
if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(crossYearResult)) {
  console.log(`PASS [zh-CN cross-year]: "${crossYearResult}" (YYYY-MM-DD HH:mm:ss format)`);
} else {
  console.log(`FAIL [zh-CN cross-year]: "${crossYearResult}" (expected YYYY-MM-DD HH:mm:ss format)`);
  failures++;
}

if (failures > 0) {
  console.log(`\n${failures} test(s) FAILED`);
  process.exit(1);
}
console.log(`\nAll tests PASSED`);
