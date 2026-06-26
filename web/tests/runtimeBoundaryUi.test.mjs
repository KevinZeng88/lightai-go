import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')
const files = [
  'src/pages/BackendRuntimesPage.vue',
  'src/pages/RunnerConfigsPage.vue',
  'src/pages/ModelDeploymentsPage.vue',
  'src/pages/BackendsPage.vue',
  'src/components/common/RuntimeParameterEditor.vue',
  'src/components/config/ConfigEditView.vue',
  'src/components/config/ConfigSection.vue',
  'src/components/config/ConfigField.vue',
  'src/utils/configEditView.ts',
  'src/components/deployments/NodeRuntimeConfigWizard.vue',
  'src/components/deployments/DeploymentOverrideEditor.vue',
  'src/api/runtimes.ts',
  'src/api/backends.ts',
  'src/api/configEdit.ts',
]

const sources = Object.fromEntries(files.map((file) => [file, fs.readFileSync(path.join(root, file), 'utf8')]))
const all = Object.values(sources).join('\n')

let failed = 0
function check(name, condition) {
  if (!condition) {
    failed += 1
    console.error(`FAIL: ${name}`)
  } else {
    console.log(`PASS: ${name}`)
  }
}

for (const token of [
  'config_snapshot_json',
  'parameter_schema_json',
  'parameter_values_json',
  'parameters_json',
  'default_args_json',
  'default_env_json',
  'docker_json',
  'model_mount_json',
  'health_check_json',
  'capabilities_json',
  'image_candidates_json',
  'default_images_json',
  'env_json',
  'ports_json',
  'volumes_json',
  'devices_json',
  'resource_controls_json',
]) {
  check(`UI does not reference ${token}`, !all.includes(token))
}

check('RuntimeParameterEditor edits ConfigSet items', sources['src/components/common/RuntimeParameterEditor.vue'].includes('config_set'))
check('RuntimeParameterEditor emits config_overrides', sources['src/components/common/RuntimeParameterEditor.vue'].includes('config_overrides'))
check('RuntimeParameterEditor supports extension labels', sources['src/components/common/RuntimeParameterEditor.vue'].includes('extensions?.label') || sources['src/components/common/RuntimeParameterEditor.vue'].includes("extensions &&"))
check('RuntimeParameterEditor supports schema ordering', sources['src/components/common/RuntimeParameterEditor.vue'].includes('order'))
check('RuntimeParameterEditor supports select controls', sources['src/components/common/RuntimeParameterEditor.vue'].includes('el-select'))
check('RuntimeParameterEditor can render fake_new_param from ConfigSet schema', sources['src/components/common/RuntimeParameterEditor.vue'].includes('fake_new_param') || sources['src/components/common/RuntimeParameterEditor.vue'].includes('itemLabel(item)'))
check('Backend runtime page no longer imports hardcoded human form', !sources['src/pages/BackendRuntimesPage.vue'].includes('HumanRuntimeParameterForm'))
check('Node runtime wizard no longer imports hardcoded human form', !sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('HumanRuntimeParameterForm'))
check('Runtime human field helper is not used by active pages', !sources['src/pages/BackendRuntimesPage.vue'].includes('getHumanFieldsForBackend') && !sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('getHumanFieldsForBackend'))
check('Backend runtime page shows Advanced Diagnostics (ConfigSet collapsed)', sources['src/pages/BackendRuntimesPage.vue'].includes('advancedDiagnostics') || sources['src/pages/BackendRuntimesPage.vue'].includes('Advanced Diagnostics'))
check('Runner config page enables NBR through current route (via wizard)', sources['src/pages/RunnerConfigsPage.vue'].includes('NodeRuntimeConfigWizard') || all.includes('/backend-runtimes/enable'))
check('Runner config page checks NBR through check-request', all.includes('check-request'))
check('Deployment create uses node_backend_runtime_id', sources['src/pages/ModelDeploymentsPage.vue'].includes('node_backend_runtime_id'))
check('Deployment create uses DeploymentWizard (with config_overrides)', sources['src/pages/ModelDeploymentsPage.vue'].includes('DeploymentWizard'))
check('Deployment edit runtime selector is absent', !sources['src/pages/ModelDeploymentsPage.vue'].includes('editForm') && !sources['src/pages/ModelDeploymentsPage.vue'].includes('runtime selector'))
check('Model deployment page does not import RuntimeParameterEditor', !sources['src/pages/ModelDeploymentsPage.vue'].includes('RuntimeParameterEditor'))
check('ConfigEditView renders sections in order', sources['src/components/config/ConfigEditView.vue'].includes('sortedSections'))
check('ConfigEditView required fields cannot be disabled', sources['src/components/config/ConfigField.vue'].includes('!field.required') && sources['src/components/config/ConfigField.vue'].includes('field.has_enable'))
check('ConfigEditView optional fields show enabled checkbox', sources['src/components/config/ConfigField.vue'].includes('el-checkbox') && sources['src/components/config/ConfigField.vue'].includes('has_enable'))
check('ConfigEditView disables input when field disabled', sources['src/components/config/ConfigField.vue'].includes('!field.enabled || readonly'))
check('ConfigEditView has structured Docker widgets', sources['src/components/config/ConfigField.vue'].includes('device_table') && sources['src/components/config/ConfigField.vue'].includes('key_value_table') && sources['src/components/config/ConfigField.vue'].includes('mount_form'))
check('ConfigEditView ordinary rendering does not show launcher.docker_options label', !sources['src/components/config/ConfigField.vue'].includes('{{ field.internal_key }}') && !sources['src/components/config/ConfigField.vue'].includes('{{ field.key }}'))
check('Advanced raw section defaults collapsed', sources['src/components/config/ConfigSection.vue'].includes('section.collapsed') && sources['src/components/config/ConfigSection.vue'].includes('advanced_raw'))
check('BackendRuntimesPage uses ConfigEditView', sources['src/pages/BackendRuntimesPage.vue'].includes('ConfigEditView') && !sources['src/pages/BackendRuntimesPage.vue'].includes('RuntimeParameterEditor'))
check('BackendsPage uses ConfigEditView for BackendVersion', sources['src/pages/BackendsPage.vue'].includes('ConfigEditView'))
check('NodeRuntimeConfigWizard uses ConfigEditView', sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('ConfigEditView') && !sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('RuntimeParameterEditor'))
check('DeploymentOverrideEditor uses ConfigEditView patch model', sources['src/components/deployments/DeploymentOverrideEditor.vue'].includes('ConfigEditView') && sources['src/components/deployments/DeploymentOverrideEditor.vue'].includes('editable_config_patch'))
check('Config edit API client uses view/apply endpoints', sources['src/api/configEdit.ts'].includes('/config-edit/view') && sources['src/api/configEdit.ts'].includes('/config-edit/apply'))
check('NodeRuntimeConfigWizard selector main title avoids raw id', !sources['src/components/deployments/NodeRuntimeConfigWizard.vue'].includes('runtime.id }}</div>'))
check('BackendRuntime clone dialog passes display_name/name', sources['src/pages/BackendRuntimesPage.vue'].includes('cloneForm') && sources['src/pages/BackendRuntimesPage.vue'].includes('display_name') && sources['src/pages/BackendRuntimesPage.vue'].includes('name'))

if (failed > 0) {
  console.error(`\n${failed} test(s) FAILED`)
  process.exit(1)
}

console.log('\nAll tests PASSED')
