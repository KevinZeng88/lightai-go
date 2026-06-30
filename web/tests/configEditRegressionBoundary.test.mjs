import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')

function read(file) {
  return fs.readFileSync(path.join(root, file), 'utf8')
}

const sources = {
  configEditView: read('src/utils/configEditView.ts'),
  configEditDisplay: read('src/utils/configEditDisplay.ts'),
  configEditViewComponent: read('src/components/config/ConfigEditView.vue'),
  configSection: read('src/components/config/ConfigSection.vue'),
  configField: read('src/components/config/ConfigField.vue'),
  backendRuntimes: read('src/pages/BackendRuntimesPage.vue'),
  runnerConfigs: read('src/pages/RunnerConfigsPage.vue'),
  deployments: read('src/pages/ModelDeploymentsPage.vue'),
  backends: read('src/pages/BackendsPage.vue'),
  nodeRuntimeWizard: read('src/components/deployments/NodeRuntimeConfigWizard.vue'),
  deploymentWizard: read('src/components/deployments/DeploymentWizard.vue'),
  deploymentOverride: read('src/components/deployments/DeploymentOverrideEditor.vue'),
}

let failed = 0
function check(name, condition) {
  if (!condition) {
    failed += 1
    console.error(`FAIL: ${name}`)
  } else {
    console.log(`PASS: ${name}`)
  }
}

const displayGroupFn = sources.configEditView.match(/export function displayGroupForField[\s\S]*?\n}/)?.[0] || ''
check(
  'displayGroupForField checks enabledAtLoad before isExpertField',
  displayGroupFn.indexOf('enabledAtLoad') >= 0 &&
    displayGroupFn.indexOf('isExpertField') >= 0 &&
    displayGroupFn.indexOf('enabledAtLoad') < displayGroupFn.indexOf('isExpertField'),
)

check('ConfigEditView stable selector exists', sources.configEditViewComponent.includes('data-testid="config-edit-view"'))
check('ConfigSection stable selector exists', sources.configSection.includes('data-testid="config-edit-section"'))
check('ConfigField stable selectors exist',
  sources.configField.includes('data-testid="config-field"') &&
  sources.configField.includes('data-testid="config-field-enabled"') &&
  sources.configField.includes('data-testid="config-field-value"'))
check('ConfigField exposes risk/tier/view/diagnostic metadata',
  sources.configField.includes(`:data-field-tier="field.tier || ''"`) &&
  sources.configField.includes(`:data-field-view="field.view || ''"`) &&
  sources.configField.includes(`:data-field-risk="field.risk || ''"`) &&
  sources.configField.includes(':data-field-diagnostic="field.diagnostic ? \'true\' : \'false\'"')
)
check('ConfigField does not disable value controls because field.enabled=false',
  !sources.configField.includes('!field.enabled || readonly') &&
  !sources.configField.includes('!field.enabled || isControlReadonly'))

for (const [name, src] of [
  ['BackendRuntimesPage', sources.backendRuntimes],
  ['RunnerConfigsPage', sources.runnerConfigs],
  ['ModelDeploymentsPage', sources.deployments],
]) {
  check(`${name} uses configEditViewLevelOptions(t)`, src.includes('configEditViewLevelOptions(t)'))
  check(`${name} uses configEditViewLevelHelp(t)`, src.includes('configEditViewLevelHelp(t)'))
  check(`${name} does not hardcode Normal/Advanced/Developer labels`,
    !src.includes("label: 'Normal'") && !src.includes("label: 'Advanced'") && !src.includes("label: 'Developer'"))
}

check('BackendRuntimesPage reloads ConfigEdit view on level changes',
  sources.backendRuntimes.includes('watch([selected, configViewLevel]') &&
  sources.backendRuntimes.includes('getConfigEditView') &&
  sources.backendRuntimes.includes('editPatch.value = null'))
check('RunnerConfigsPage reloads ConfigEdit view on level changes',
  sources.runnerConfigs.includes('watch([selected, configViewLevel]') &&
  sources.runnerConfigs.includes('getConfigEditView') &&
  sources.runnerConfigs.includes('nbrEditPatch.value = null'))
check('ModelDeploymentsPage reloads ConfigEdit view on level changes',
  sources.deployments.includes('watch(configViewLevel') &&
  sources.deployments.includes('editing.value') &&
  sources.deployments.includes('getConfigEditView') &&
  sources.deployments.includes('deploymentEditPatch.value = null'))

check('BackendRuntimesPage raw diagnostics only visible in developer mode',
  sources.backendRuntimes.includes("v-if=\"configViewLevel === 'developer'\""))
check('RunnerConfigsPage raw diagnostics only visible in developer mode',
  sources.runnerConfigs.includes("v-if=\"configViewLevel === 'developer'\""))
check('ModelDeploymentsPage raw diagnostics only visible in developer mode',
  sources.deployments.includes("v-if=\"configViewLevel === 'developer'\""))

const activeSources = [
  sources.backendRuntimes,
  sources.runnerConfigs,
  sources.deployments,
  sources.backends,
  sources.nodeRuntimeWizard,
  sources.deploymentWizard,
  sources.deploymentOverride,
].join('\n')
check('RuntimeParameterEditor is not used by active ConfigEdit pages', !activeSources.includes('RuntimeParameterEditor'))
check('HumanRuntimeParameterForm is not used by active ConfigEdit pages', !activeSources.includes('HumanRuntimeParameterForm'))

if (failed > 0) {
  console.error(`\n${failed} ConfigEdit regression boundary test(s) FAILED`)
  process.exit(1)
}

console.log('\nConfigEdit regression boundary tests PASSED')
