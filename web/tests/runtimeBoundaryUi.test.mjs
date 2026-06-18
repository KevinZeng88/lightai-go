import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(new URL('.', import.meta.url).pathname, '..')
const templatePage = fs.readFileSync(path.join(root, 'src/pages/BackendRuntimesPage.vue'), 'utf8')
const runnerPage = fs.readFileSync(path.join(root, 'src/pages/RunnerConfigsPage.vue'), 'utf8')

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

if (failed > 0) {
  process.exit(1)
}
