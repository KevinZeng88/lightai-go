import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')
const templatePage = fs.readFileSync(path.join(root, 'src/pages/BackendRuntimesPage.vue'), 'utf8')
const runnerPage = fs.readFileSync(path.join(root, 'src/pages/RunnerConfigsPage.vue'), 'utf8')
const deploymentsPage = fs.readFileSync(path.join(root, 'src/pages/ModelDeploymentsPage.vue'), 'utf8')
const artifactsPage = fs.readFileSync(path.join(root, 'src/pages/ModelArtifactsPage.vue'), 'utf8')
const instancesPage = fs.readFileSync(path.join(root, 'src/pages/ModelInstancesPage.vue'), 'utf8')
const layout = fs.readFileSync(path.join(root, 'src/layouts/ConsoleLayout.vue'), 'utf8')
const runtimeParamEditor = fs.readFileSync(path.join(root, 'src/components/common/RuntimeParameterEditor.vue'), 'utf8')

let failed = 0
function check(name, condition) {
  if (!condition) {
    failed += 1
    console.error(`FAIL: ${name}`)
  } else {
    console.log(`PASS: ${name}`)
  }
}

// --- Existing structural checks ---
check('runtime template page does not expose add-node action', !templatePage.includes('showAddNode') && !templatePage.includes('doAddNode'))
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
check('instance test dialog supports mode selection', instancesPage.includes('testMode') && instancesPage.includes('value="chat"') && instancesPage.includes('value="completion"'))
check('model artifact page displays inferred capabilities', artifactsPage.includes('inferModelCapabilities') && artifactsPage.includes('capabilityReadonlyHint') && artifactsPage.includes('recommendedEndpoint'))

// --- OOM prevention: syncing guard with try/finally ---
check('RuntimeParameterEditor has try/finally syncing guard', runtimeParamEditor.includes('let syncing = false') && runtimeParamEditor.includes('syncing = true') && runtimeParamEditor.includes('finally'))

// --- CRITICAL: Value editors always visible, never hidden by v-if ---
// RuntimeParameterEditor must NOT use v-if to hide inputs when disabled
check('RuntimeParameterEditor does NOT hide scalar inputs with v-if when disabled', !runtimeParamEditor.includes('v-if="opt.enabled"'))
check('RuntimeParameterEditor uses :disabled instead of v-if for scalar inputs', runtimeParamEditor.includes(':disabled="!opt.enabled'))
check('RuntimeParameterEditor always shows textarea for list options', !runtimeParamEditor.match(/v-if.*enabled.*\n.*textarea/))

// BackendRuntimesPage must NOT use v-if to hide inputs
// BackendRuntimesPage delegates to RuntimeParameterEditor (no inline duplicate params)
check('BackendRuntimesPage does NOT have inline scalarOptions (delegated to RPE)', !templatePage.includes('const scalarOptions = reactive'))
check('BackendRuntimesPage uses RuntimeParameterEditor for all params', templatePage.includes('<RuntimeParameterEditor'))
check('BackendRuntimesPage showEdit loads into parameterEditorModel', templatePage.includes('parameterEditorModel.value ='))

// --- Values preserved: buildPayload reads from parameterEditorModel ---
check('BackendRuntimesPage buildPayload reads from parameterEditorModel', templatePage.includes('const m = parameterEditorModel.value'))

// --- Clone preserves all values via cloneParameterEditorModel ---
check('BackendRuntimesPage showClone uses cloneParameterEditorModel', templatePage.includes('cloneParameterEditorModel.value ='))

// --- RunnerConfigsPage: NO legacy Docker editor (single-entry via RuntimeParameterEditor) ---
// Docker --privileged explanation is now in RuntimeParameterEditor's risk label
check('RuntimeParameterEditor privileged has risk warning', runtimeParamEditor.includes('runtimes.privilegedRisk') || runtimeParamEditor.includes('Docker --privileged'))
// Negative assertions: legacy inline Docker fields must NOT exist (no duplicate entries)
check('runner config does NOT have legacy editDevicesText', !runnerPage.includes('editDevicesText'))
check('runner config does NOT have legacy editGroupAddText', !runnerPage.includes('editGroupAddText'))
check('runner config does NOT have legacy editSecurityOptText', !runnerPage.includes('editSecurityOptText'))
check('runner config does NOT have legacy editShmSize', !runnerPage.includes('editShmSize'))
check('runner config does NOT have legacy editUlimitsText', !runnerPage.includes('editUlimitsText'))
check('runner config does NOT have legacy editPrivileged', !runnerPage.includes('editPrivileged'))
check('runner config does NOT have legacy editIpcMode', !runnerPage.includes('editIpcMode'))
// Positive: RunnerConfigsPage delegates to RuntimeParameterEditor
check('runner config uses RuntimeParameterEditor', runnerPage.includes('RuntimeParameterEditor'))
check('runner config populates editParameterModel', runnerPage.includes('editParameterModel.value'))
// Positive: non-Docker convenience editors still present
check('runner config edit has volumes textarea', runnerPage.includes('editVolumesText'))
check('runner config edit has ports textarea', runnerPage.includes('editPortsText'))
check('runner config edit has env textarea', runnerPage.includes('editEnvText'))

// --- Full parameter coverage in RuntimeParameterEditor ---
check('RuntimeParameterEditor has privileged', runtimeParamEditor.includes("key: 'privileged'"))
check('RuntimeParameterEditor has ipc_mode', runtimeParamEditor.includes("key: 'ipc_mode'"))
check('RuntimeParameterEditor has uts_mode', runtimeParamEditor.includes("key: 'uts_mode'"))
check('RuntimeParameterEditor has network_mode', runtimeParamEditor.includes("key: 'network_mode'"))
check('RuntimeParameterEditor has pid_mode', runtimeParamEditor.includes("key: 'pid_mode'"))
check('RuntimeParameterEditor has shm_size', runtimeParamEditor.includes("key: 'shm_size'"))
check('RuntimeParameterEditor has devices', runtimeParamEditor.includes("key: 'devices'"))
check('RuntimeParameterEditor has group_add', runtimeParamEditor.includes("key: 'group_add'"))
check('RuntimeParameterEditor has security_options', runtimeParamEditor.includes("key: 'security_options'"))
check('RuntimeParameterEditor has cap_add', runtimeParamEditor.includes("key: 'cap_add'"))
check('RuntimeParameterEditor has device_cgroup_rules', runtimeParamEditor.includes("key: 'device_cgroup_rules'"))
check('RuntimeParameterEditor has extra_hosts', runtimeParamEditor.includes("key: 'extra_hosts'"))
check('RuntimeParameterEditor has ulimits', runtimeParamEditor.includes("key: 'ulimits'"))
check('RuntimeParameterEditor has extra_mounts', runtimeParamEditor.includes("key: 'extra_mounts'"))

// --- Command preview includes all parameters ---
check('command preview includes --privileged', runtimeParamEditor.includes("parts.push('--privileged'"))
check('command preview includes --ipc', runtimeParamEditor.includes("parts.push('--ipc'"))
check('command preview includes --uts', runtimeParamEditor.includes("parts.push('--uts'"))
check('command preview includes --network', runtimeParamEditor.includes("parts.push('--network'"))
check('command preview includes --pid', runtimeParamEditor.includes("parts.push('--pid'"))
check('command preview includes --shm-size', runtimeParamEditor.includes("parts.push('--shm-size'"))
check('command preview includes --device', runtimeParamEditor.includes("parts.push('--device'"))
check('command preview includes --group-add', runtimeParamEditor.includes("parts.push('--group-add'"))
check('command preview includes --security-opt', runtimeParamEditor.includes("parts.push('--security-opt'"))
check('command preview includes --cap-add', runtimeParamEditor.includes("parts.push('--cap-add'"))
check('command preview includes --ulimit', runtimeParamEditor.includes("parts.push('--ulimit'"))
check('command preview includes --add-host', runtimeParamEditor.includes("parts.push('--add-host'"))

// --- Parameter layer separation: Model page must NOT show Docker runtime params ---
check('Model page does NOT import RuntimeParameterEditor', !artifactsPage.includes("import RuntimeParameterEditor"))
check('Model page does NOT show Docker runtime param editor', !artifactsPage.includes('<RuntimeParameterEditor'))
check('Model page shows model-specific serving param hints', artifactsPage.includes('parameterDefaultsText') && artifactsPage.includes('parameterDefaultsHint'))

// --- Dynamic backend schema support ---
check('RuntimeParameterEditor accepts backendSchema prop', runtimeParamEditor.includes('backendSchema'))
check('RuntimeParameterEditor has backend serving args section', runtimeParamEditor.includes('backendServingArgs') || runtimeParamEditor.includes('backendArgs'))
check('RuntimeParameterEditor renders backend params dynamically', runtimeParamEditor.includes('backendParams') && runtimeParamEditor.includes('v-for'))
check('BackendRuntimesPage passes backendSchema to editor', templatePage.includes('backend-schema') || templatePage.includes(':backendSchema'))
check('BackendRuntimesPage loads backend version schema', templatePage.includes('loadBackendSchema') || templatePage.includes('default_args_schema_json'))

// --- Preflight error display ---
check('Preflight errors show error code', deploymentsPage.includes('e.code'))
check('Preflight errors show all error codes including format_mismatch', deploymentsPage.includes('format_mismatch'))
check('Preflight errors show context details', deploymentsPage.includes('preflightErrorContext'))

// --- Deployment override passes backendSchema to RuntimeParameterEditor ---
check('Deployment edit passes backend-schema to editor', deploymentsPage.includes(':backend-schema="editBackendSchema"'))
check('Deployment edit loads backend schema async', deploymentsPage.includes('loadEditBackendSchema'))

// --- BackendRuntimesPage uses parameterEditorModel (no duplicate inline state) ---
check('BackendRuntimesPage buildPayload reads from parameterEditorModel', templatePage.includes('const m = parameterEditorModel.value'))
check('BackendRuntimesPage showEdit sets parameterEditorModel', templatePage.includes('parameterEditorModel.value ='))

// --- Vendor isolation: NVIDIA runtimes have no devices in catalog ---
// This is a code-level check: the nvidia-docker.yaml files should NOT have a 'devices' key

if (failed > 0) {
  console.error(`\n${failed} test(s) FAILED`)
  process.exit(1)
} else {
  console.log(`\nAll tests PASSED`)
}
