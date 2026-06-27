import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { transform } from 'esbuild'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')

async function importTs(relativePath) {
  const source = fs.readFileSync(path.join(root, relativePath), 'utf8')
  const result = await transform(source, {
    loader: 'ts',
    format: 'esm',
    target: 'es2022',
  })
  const encoded = Buffer.from(result.code).toString('base64')
  return import(`data:text/javascript;base64,${encoded}`)
}

function baseView(field) {
  return {
    layer: 'node_backend_runtime',
    object_id: 'nbr-test',
    object_kind: 'node_backend_runtime',
    sections: [
      {
        key: 'model_serving',
        label: 'Model Serving',
        order: 1,
        fields: [field],
      },
    ],
  }
}

function field(overrides = {}) {
  return {
    key: 'model_runtime.max_model_len',
    internal_key: 'model_runtime.max_model_len',
    label: 'Max model length',
    section: 'model_serving',
    order: 1,
    type: 'integer',
    widget: 'number',
    value: 4096,
    enabled: false,
    has_enable: true,
    required: false,
    readonly: false,
    advanced: false,
    original_value: 4096,
    original_enabled: false,
    ...overrides,
  }
}

const { buildConfigEditPatch } = await importTs('src/utils/configEditView.ts')

{
  const patch = buildConfigEditPatch(baseView(field({ enabled: true })))
  assert.equal(patch.fields.length, 1)
  assert.equal(patch.fields[0].enabled, true)
  assert.equal(patch.fields[0].value, 4096)
}

{
  const patch = buildConfigEditPatch(baseView(field({ value: 8192 })))
  assert.equal(patch.fields.length, 1)
  assert.equal(patch.fields[0].enabled, false)
  assert.equal(patch.fields[0].value, 8192)
}

{
  const patch = buildConfigEditPatch(baseView(field({ enabled: false, original_enabled: true })))
  assert.equal(patch.fields.length, 1)
  assert.equal(patch.fields[0].enabled, false)
}

{
  const patch = buildConfigEditPatch(baseView(field()))
  assert.equal(patch.fields.length, 0)
}

{
  const patch = buildConfigEditPatch(baseView(field({
    required: true,
    has_enable: false,
    enabled: false,
    original_enabled: false,
  })))
  assert.equal(patch.fields.length, 1)
  assert.equal(patch.fields[0].enabled, true)
}

const fieldSource = fs.readFileSync(path.join(root, 'src/components/config/ConfigField.vue'), 'utf8')
assert.equal(fieldSource.includes('!field.enabled || readonly'), false, 'ConfigField value controls must not be disabled only because field.enabled=false')
assert.equal(fieldSource.includes('v-model="field.enabled"'), true, 'ConfigField must expose independent enabled toggle')
assert.equal(fieldSource.includes('v-model="field.value"'), true, 'ConfigField must keep independent value binding')
assert.equal(fieldSource.includes('data-testid="config-field"'), true, 'ConfigField must expose stable field selector')
assert.equal(fieldSource.includes('data-testid="config-field-enabled"'), true, 'ConfigField must expose stable enabled selector')
assert.equal(fieldSource.includes('data-testid="config-field-value"'), true, 'ConfigField must expose stable value selector')
assert.equal(fieldSource.includes(':data-field-key="field.key"'), true, 'ConfigField selectors must include field key')
assert.equal(fieldSource.includes(':data-internal-key="field.internal_key"'), true, 'ConfigField selectors must include internal key')

const viewSource = fs.readFileSync(path.join(root, 'src/components/config/ConfigEditView.vue'), 'utf8')
assert.equal(viewSource.includes('data-testid="config-edit-view"'), true, 'ConfigEditView must expose stable root selector')
assert.equal(viewSource.includes(':data-layer="localView.layer"'), true, 'ConfigEditView selector must include layer')
assert.equal(viewSource.includes(':data-object-id="localView.object_id"'), true, 'ConfigEditView selector must include object id')

const sectionSource = fs.readFileSync(path.join(root, 'src/components/config/ConfigSection.vue'), 'utf8')
assert.equal(sectionSource.includes('data-testid="config-edit-section"'), true, 'ConfigSection must expose stable section selector')
assert.equal(sectionSource.includes(':data-section-key="section.key"'), true, 'ConfigSection selector must include section key')

console.log('ConfigEdit contract tests PASSED')
