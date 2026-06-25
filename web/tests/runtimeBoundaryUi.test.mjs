import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')
const files = [
  'src/pages/BackendRuntimesPage.vue',
  'src/pages/RunnerConfigsPage.vue',
  'src/pages/ModelDeploymentsPage.vue',
  'src/pages/BackendsPage.vue',
  'src/components/common/RuntimeParameterEditor.vue',
  'src/api/runtimes.ts',
  'src/api/backends.ts',
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
check('Backend runtime page shows ConfigSet', sources['src/pages/BackendRuntimesPage.vue'].includes('ConfigSet'))
check('Runner config page enables NBR through current route', sources['src/pages/RunnerConfigsPage.vue'].includes('/backend-runtimes/enable'))
check('Runner config page checks NBR through check-request', sources['src/pages/RunnerConfigsPage.vue'].includes('check-request'))
check('Deployment create uses node_backend_runtime_id', sources['src/pages/ModelDeploymentsPage.vue'].includes('node_backend_runtime_id'))
check('Deployment create uses config_overrides', sources['src/pages/ModelDeploymentsPage.vue'].includes('config_overrides'))
check('Deployment edit runtime selector is absent', !sources['src/pages/ModelDeploymentsPage.vue'].includes('editForm') && !sources['src/pages/ModelDeploymentsPage.vue'].includes('runtime selector'))
check('Model deployment page does not import RuntimeParameterEditor', !sources['src/pages/ModelDeploymentsPage.vue'].includes('RuntimeParameterEditor'))

if (failed > 0) {
  console.error(`\n${failed} test(s) FAILED`)
  process.exit(1)
}

console.log('\nAll tests PASSED')
