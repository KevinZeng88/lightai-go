import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')
const files = [
  'src/pages/BackendRuntimesPage.vue',
  'src/pages/RunnerConfigsPage.vue',
  'src/pages/ModelDeploymentsPage.vue',
  'src/pages/BackendsPage.vue',
  'src/components/config/ConfigEditView.vue',
  'src/components/config/ConfigSection.vue',
  'src/components/config/ConfigField.vue',
  'src/utils/configEditView.ts',
  'src/components/deployments/NodeRuntimeConfigWizard.vue',
  'src/components/deployments/DeploymentWizard.vue',
  'src/components/DockerImagePicker.vue',
  'src/components/deployments/DeploymentOverrideEditor.vue',
  'src/api/runtimes.ts',
  'src/api/backends.ts',
  'src/api/configEdit.ts',
  'src/utils/runtimeDisplay.ts',
  'src/locales/zh-CN.ts',
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

// RuntimeParameterEditor was intentionally removed (OI-07) — replaced by ConfigEditView.
// HumanRuntimeParameterForm was intentionally removed — replaced by ConfigEditView.
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
check('ConfigEditView keeps inputs editable when field disabled', !sources['src/components/config/ConfigField.vue'].includes('!field.enabled || readonly'))
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

// -- Follow-up repair tests (2026-06-27) --

// a. Section key i18n: ConfigSection maps keys to i18n, not raw English labels.
const sectionSrc = sources['src/components/config/ConfigSection.vue']
check('ConfigSection maps section.key to i18n (SECTION_I18N_MAP)', sectionSrc.includes('SECTION_I18N_MAP'))
check('ConfigSection does not show raw section.label directly', !sectionSrc.includes('{{ section.label }}'))
check('ConfigSection uses sectionI18nLabel computed', sectionSrc.includes('sectionI18nLabel'))
check('ConfigSection advanced_raw tag uses i18n', sectionSrc.includes('configEdit.sections.advancedRaw'))

// b. Field label i18n: ConfigField maps field.key to configEdit.labels.
const fieldSrc = sources['src/components/config/ConfigField.vue']
check('ConfigField has displayLabel computed with i18n mapping', fieldSrc.includes('displayLabel') && fieldSrc.includes('configEdit.labels'))
check('ConfigField template uses displayLabel not field.label', fieldSrc.includes('{{ displayLabel }}'))

// c. key_value_table has editable key column.
check('key_value_table key column has el-input', fieldSrc.includes('key_value_table') && fieldSrc.includes('v-model="row.key"'))
check('key_value_table filters empty keys on writeback', fieldSrc.includes('.filter((r') && fieldSrc.includes('r.key.trim()'))
check('key_value_table has both key and value columns', fieldSrc.includes('v-model="row.key"') && fieldSrc.includes('v-model="row.value"'))

// d. device_table editable columns.
check('device_table host_path has el-input', fieldSrc.includes('device_table') && fieldSrc.includes('v-model="row.host_path"'))
check('device_table container_path has el-input', fieldSrc.includes('v-model="row.container_path"'))
check('device_table readonly has el-switch', fieldSrc.includes('row.readonly') && fieldSrc.includes('v-model="row.readonly"'))

// e. optional_devices string array handling.
check('device_table initDeviceRows handles string arrays (allStrings)', fieldSrc.includes('allStrings'))
check('device_table onDeviceTableChange preserves readonly', fieldSrc.includes('d.readonly'))

// f. runtimeDisplay extractVersion returns * for all runtimes (no per-version differentiated configs).
const rtSrc = sources['src/utils/runtimeDisplay.ts']
check('runtimeDisplay extractVersion returns * unconditionally', rtSrc.includes("return '*'"))
check('runtimeDisplay does not show per-version number to user', !rtSrc.includes('v0.23.0') && !rtSrc.includes('sglang-v0.5.13') && !rtSrc.includes('llamacpp-b9700'))
// g. runtimeDisplay normalizes runtime.xxx prefix names.
check('runtimeDisplay strips runtime. prefix from display_name', rtSrc.includes("replace(/^runtime\\./, '')"))
check('runtimeDisplay strips runtime. prefix from name', rtSrc.includes('normalizedName'))

// h. Config edit scope: RunnerConfigsPage migrated from RuntimeParameterEditor to ConfigEditView.
const rcpSrc = sources['src/pages/RunnerConfigsPage.vue']
check('RunnerConfigsPage uses ConfigEditView', rcpSrc.includes('ConfigEditView'))
check('RunnerConfigsPage does NOT use RuntimeParameterEditor', !rcpSrc.includes('RuntimeParameterEditor'))
check('RunnerConfigsPage applies editable_config_patch', rcpSrc.includes('applyConfigEditPatch'))
check('RunnerConfigsPage uses node_backend_runtime layer', rcpSrc.includes('node_backend_runtime'))
check('RunnerConfigsPage has NBR delete confirmation', rcpSrc.includes('ElMessageBox') && rcpSrc.includes('deleteNBR'))
check('RunnerConfigsPage deletes NBR through node-scoped route', rcpSrc.includes('apiClient.delete(`/nodes/${row.node_id}/backend-runtimes/${row.id}`)'))

// i. NBR wizard has node image selector.
const wizardSrc = sources['src/components/deployments/NodeRuntimeConfigWizard.vue']
check('NodeRuntimeConfigWizard uses Docker image picker with manual input support', wizardSrc.includes('DockerImagePicker'))
check('NodeRuntimeConfigWizard no longer owns ad hoc node image loading', !wizardSrc.includes('loadNodeImages'))
check('NodeRuntimeConfigWizard uses node_backend_runtime layer for ConfigEditView', wizardSrc.includes('node_backend_runtime'))
check('NodeRuntimeConfigWizard uses DockerImagePicker component', wizardSrc.includes('DockerImagePicker'))
check('NodeRuntimeConfigWizard does not read file-style repoTags image fields', !wizardSrc.includes('repoTags'))
const pickerSrc = sources['src/components/DockerImagePicker.vue']
check('DockerImagePicker does not render NaN image sizes', pickerSrc.includes('Number.isFinite') && !pickerSrc.includes('NaN KB'))

// j. BackendsPage has developer i18n for Add Parameter.
const beSrc = sources['src/pages/BackendsPage.vue']
check('BackendsPage Add Parameter uses i18n title', beSrc.includes('addParameter'))
check('BackendsPage Add Parameter has developer hint', beSrc.includes('addParameterHint'))

// k. Runtime name normalization.
check('runtimeDisplay normalizes runtime.xxx prefix names', rtSrc.includes('normalizedDisplay'))
check('runtimeDisplay has product-friendly backend/vendor maps', rtSrc.includes('BACKEND_DISPLAY') && rtSrc.includes('VENDOR_DISPLAY'))
// k2. Tech slug detection for product-friendly names.
check('runtimeDisplay detects tech slugs and uses product name', rtSrc.includes('techSlugPattern'))
check('runtimeDisplay has displayIsTechSlug check', rtSrc.includes('displayIsTechSlug'))

// l. Canonical alias i18n keys present in zh-CN.
const zhSrc = sources['src/locales/zh-CN.ts']
check('zh-CN has service.listen_host i18n', zhSrc.includes('service.listen_host'))
check('zh-CN has service.container_port i18n', zhSrc.includes('service.listen_host') && zhSrc.includes('容器监听端口'))

// m. Deployment wizard uses complete model location eligibility.
const deploymentWizardSrc = sources['src/components/deployments/DeploymentWizard.vue']
check('DeploymentWizard falls back to artifact.locations', deploymentWizardSrc.includes('selectedArtifact') && deploymentWizardSrc.includes('locations'))
check('DeploymentWizard checks deployable match_status', deploymentWizardSrc.includes('match_status') && deploymentWizardSrc.includes('exact_match') && deploymentWizardSrc.includes('probable_match') && deploymentWizardSrc.includes('manual_attested'))
check('DeploymentWizard compatibility error includes artifact and node context', deploymentWizardSrc.includes('model_artifact_id') && deploymentWizardSrc.includes('nbrNodeId') && deploymentWizardSrc.includes('visibleLocations'))

	// n. Object child field display: ConfigField handles structured widgets.
	check('ConfigField has mount_form widget for model_mount', fieldSrc.includes('mount_form'))
	check('ConfigField has health_check_form widget for health', fieldSrc.includes('health_check_form'))
	check('ConfigField has key_value_table widget for env', fieldSrc.includes('key_value_table'))
	check('ConfigField handles null/absent values with fallback display', fieldSrc.includes('v === null || v === undefined'))
	check('ConfigField formattedDisplayValue shows - for null', fieldSrc.includes("return '-'"))

	// o. Raw Config Set JSON collapsed by default in detail pages.
	check('BackendRuntimesPage raw ConfigSet is in diagnostics collapse',
		sources['src/pages/BackendRuntimesPage.vue'].includes('advancedDiagnostics') &&
		sources['src/pages/BackendRuntimesPage.vue'].includes('el-collapse'))
	check('RunnerConfigsPage raw probe evidence is collapsed by default',
		sources['src/pages/RunnerConfigsPage.vue'].includes('el-collapse') &&
		sources['src/pages/RunnerConfigsPage.vue'].includes('probe_results_json'))

if (failed > 0) {
  console.error(`\n${failed} test(s) FAILED`)
  process.exit(1)
}

console.log('\nAll tests PASSED')
