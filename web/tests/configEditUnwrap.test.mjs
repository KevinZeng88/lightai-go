import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')

// Test 1: Source-code verify getConfigEditView unwraps config_edit_view envelope.
// The backend returns { config_edit_view, config_view }.  The frontend must
// return the inner config_edit_view object (not the envelope) so that
// ConfigEditView.vue receives an object with a .sections property.

const apiSrc = fs.readFileSync(path.join(root, 'src/api/configEdit.ts'), 'utf8')

// Unwrap pattern must exist
assert.ok(
  apiSrc.includes('config_edit_view'),
  'getConfigEditView must reference config_edit_view for envelope unwrap'
)

// Must have fallback to raw response for backward compatibility
assert.ok(
  apiSrc.includes('?? resp'),
  'getConfigEditView must have ?? resp fallback for non-envelope responses'
)

// Verify the return type is ConfigEditView (not the envelope)
assert.ok(
  apiSrc.includes('Promise<ConfigEditView>'),
  'getConfigEditView must declare return type ConfigEditView'
)

// The function must NOT return the raw envelope directly
const returnLine = apiSrc.split('\n').find(l => l.includes('return') && l.includes('resp'))
assert.ok(
  returnLine && (returnLine.includes('config_edit_view') || returnLine.includes('??')),
  'getConfigEditView return must unwrap config_edit_view, not return raw envelope'
)

// Verify the import includes ConfigEditView type
assert.ok(
  apiSrc.includes("import type { ConfigEditPatch, ConfigEditView } from '@/utils/configEditView'"),
  'configEdit.ts must import ConfigEditView type'
)

console.log('PASS: getConfigEditView unwrap contract tests')
