import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')
const templatePage = fs.readFileSync(path.join(root, 'src/pages/BackendRuntimesPage.vue'), 'utf8')
const runnerPage = fs.readFileSync(path.join(root, 'src/pages/RunnerConfigsPage.vue'), 'utf8')
const deploymentsPage = fs.readFileSync(path.join(root, 'src/pages/ModelDeploymentsPage.vue'), 'utf8')
const artifactsPage = fs.readFileSync(path.join(root, 'src/pages/ModelArtifactsPage.vue'), 'utf8')
const instancesPage = fs.readFileSync(path.join(root, 'src/pages/ModelInstancesPage.vue'), 'utf8')
const layout = fs.readFileSync(path.join(root, 'src/layouts/ConsoleLayout.vue'), 'utf8')

let failed = 0
function check(name, condition) {
  if (!condition) {
    failed += 1
    console.error(`FAIL: ${name}`)
  } else {
    console.log(`PASS: ${name}`)
  }
}

check('runtime template page does not expose add-node action', !templatePage.includes('showAddNode') && !templatePage.includes('doAddNode') && !templatePage.includes('nodeRuntime.addNode'))
check('runtime template page shows read-only usage references', templatePage.includes('usageRefsReadonly'))
check('runner config page exposes create action', runnerPage.includes('newConfig') && runnerPage.includes('startWizard'))
check('runner config page exposes edit action', runnerPage.includes('showEdit(row)') && runnerPage.includes('doEdit'))
check('runner config page exposes check action', runnerPage.includes('checkRow(row)'))
check('runner config wizard displays selected image', runnerPage.includes('dockerImages.selectedImage') && runnerPage.includes('next-summary'))
check('runtime wizard exposes name and display-name inputs', templatePage.includes('createForm.name') && templatePage.includes('createForm.display_name'))
check('runtime clone/detail shows source template', templatePage.includes('runtimes.sourceTemplate') && templatePage.includes('source_template_name'))
check('runner config wizard persists custom display name', runnerPage.includes('wizConfigName') && runnerPage.includes('display_name: wizConfigName.value'))
check('model artifact page edits display name separately from artifact name', artifactsPage.includes('form.display_name') && artifactsPage.includes('row.display_name || row.name'))
check('deployment final step has save, save-and-run, and preview actions', deploymentsPage.includes('doWizardSave') && deploymentsPage.includes('doWizardStart') && deploymentsPage.includes('doWizardPreview'))
check('deployment port labels distinguish host, container, and app ports', deploymentsPage.includes('deployments.hostPort') && deploymentsPage.includes('deployments.containerPort') && deploymentsPage.includes('deployments.appPort'))
check('deployment run button is guarded for active states', deploymentsPage.includes('isRunBlocked') && deploymentsPage.includes("'running'") && deploymentsPage.includes("'starting'"))
check('model test empty response is rendered as failure reason', instancesPage.includes('empty_model_response') && instancesPage.includes('instances.testReasonEmptyResponse'))
check('main navigation exposes model workflow group', layout.includes('nav.aiWorkflow') && layout.includes('nav.modelLibrary') && layout.includes('instances.title'))
check('backend and runtime templates are under configuration group', layout.includes('nav.config') && layout.indexOf('/backends') > layout.indexOf('nav.config') && layout.indexOf('/runtimes') > layout.indexOf('nav.config'))
check('runner config page uses structured sections before advanced JSON', runnerPage.includes('sectionImageCommand') && runnerPage.includes('sectionDevicesSecurity') && runnerPage.includes('advancedJson'))
check('instance list defaults to hiding stopped rows', instancesPage.includes('visibleItems') && instancesPage.includes("it.actual_state !== 'stopped'") && instancesPage.includes('showStopped'))
check('instance test dialog supports mode selection', instancesPage.includes('testMode') && instancesPage.includes('value=\"chat\"') && instancesPage.includes('value=\"completion\"'))
check('model artifact page displays inferred capabilities', artifactsPage.includes('inferModelCapabilities') && artifactsPage.includes('capabilityReadonlyHint') && artifactsPage.includes('recommendedEndpoint'))

if (failed > 0) {
  process.exit(1)
}
