<template>
  <div class="md-page">
    <div class="page-header">
      <h2>{{ t('modelDeployments.title') }} ({{ items.length }})</h2>
      <div class="header-actions">
        <el-button type="primary" size="small" @click="openCreate">{{ t('modelDeployments.create') }}</el-button>
        <el-button size="small" @click="refresh" :icon="RefreshRight">{{ t('common.refresh') }}</el-button>
      </div>
    
    <!-- Quick Deploy -->
    <el-collapse style="margin-bottom:12px">
      <el-collapse-item title="快速部署 (Quick Deploy)" name="quick">
        <el-form :model="qd" label-width="120px" size="small" inline>
          <el-form-item label="预设">
            <el-select v-model="qd.preset" @change="applyPreset" style="width:240px">
              <el-option label="llama.cpp CUDA + NVIDIA (GGUF) - Local E2E Example" value="llama-cpp-nvidia-local" />
              <el-option label="llama.cpp CUDA + NVIDIA (GGUF) - Custom" value="llama-cpp-nvidia-custom" />
              <el-option label="MetaX / 沐曦 Docker - Custom" value="metax-docker-custom" />
              <el-option label="Generic Docker" value="generic-docker" />
            </el-select>
          </el-form-item>
          <el-form-item label="模型路径">
            <el-input v-model="qd.modelPath" placeholder="e.g. /data/models/model.gguf (E2E example: /home/kzeng/models/...)" style="width:360px" />
          </el-form-item>
          <el-form-item label="端口">
            <el-input-number v-model="qd.hostPort" :min="1024" :max="65535" />
          </el-form-item>
        </el-form>
        <el-form v-if="qd.preset" :model="qd" label-width="120px" size="small" style="margin-top:8px">
          <el-form-item label="Node" v-if="nodes.length">
            <el-select v-model="qd.nodeId" filterable @change="onNodeChange" style="width:240px">
              <el-option v-for="n in nodes" :key="n.id" :label="n.hostname + ' (' + n.status + ')'" :value="n.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="GPU" v-if="qd.nodeId">
            <el-select v-model="qd.gpuId" style="width:240px">
              <el-option v-for="g in nodeGpus" :key="g.id" :label="g.name + ' [idx=' + g.index + '] ' + formatGB(g.memory_free_bytes || g.memory_total_bytes) + ' free'" :value="g.id" :disabled="g.health !== 'healthy'" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="quickDeploy" :loading="quickDeploying">一键部署</el-button>
          </el-form-item>
        </el-form>
        <div v-if="qd.summary" style="margin-top:8px;font-family:monospace;font-size:12px;background:#f5f5f5;padding:8px;border-radius:4px">
          <div v-for="(v,k) in qd.summary" :key="k"><b>{{ k }}:</b> {{ v }}</div>
        </div>
      </el-collapse-item>
    </el-collapse>

    </div>
    <el-table :data="items" v-loading="loading" size="small" @row-click="openDetail" highlight-current-row>
      <el-table-column :label="t('modelDeployments.name')" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column :label="t('modelDeployments.status')" width="100">
        <template #default="{ row }">
          <StatusTag :status="row.status" />
        </template>
      </el-table-column>
      <el-table-column prop="node_id" :label="t('modelDeployments.node')" width="120" show-overflow-tooltip />
      <el-table-column :label="t('modelDeployments.gpuIds')" width="120">
        <template #default="{ row }">{{ (row.gpu_ids || []).join(',') || '-' }}</template>
      </el-table-column>
      <el-table-column prop="host_port" :label="t('modelDeployments.hostPort')" width="100" />
      <el-table-column :label="t('modelDeployments.createdAt')" width="160">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="290" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click.stop="openDryRun(row)">{{ t('modelDeployments.dryRun') }}</el-button>
          <el-button size="small" text type="primary" @click.stop="handleStart(row)" :disabled="!canStart(row)">{{ t('modelDeployments.start') }}</el-button>
          <el-button size="small" text type="warning" @click.stop="handleStop(row)" :disabled="!canStop(row)">{{ t('modelDeployments.stop') }}</el-button>
          <el-button size="small" text @click.stop="$router.push(`/models/instances?deployment_id=${row.id}`)">{{ t('modelDeployments.viewInstances') }}</el-button>
        </template>
      </el-table-column>
      <template #empty><el-empty :description="t('modelDeployments.noData')" /></template>
    </el-table>

    <!-- Create Dialog -->
    <el-dialog v-model="dialogVisible" :title="t('modelDeployments.create')" width="560px" @close="resetForm">
      <el-form :model="form" label-width="140px" size="small">
        <el-form-item :label="t('modelDeployments.name')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('modelDeployments.modelArtifact')">
          <el-select v-model="form.model_artifact_id" filterable style="width:100%">
            <el-option v-for="a in artifacts" :key="a.id" :label="a.name" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelDeployments.runtimeEnvironment')">
          <el-select v-model="form.runtime_environment_id" filterable style="width:100%">
            <el-option v-for="r in runtimes" :key="r.id" :label="r.name" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelDeployments.runTemplate')">
          <el-select v-model="form.run_template_id" filterable style="width:100%">
            <el-option v-for="tpl in templates" :key="tpl.id" :label="tpl.name" :value="tpl.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelDeployments.node')">
          <el-select v-model="form.node_id" filterable style="width:100%">
            <el-option v-for="n in nodes" :key="n.id" :label="n.hostname" :value="n.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('modelDeployments.gpuIds')">
          <el-input v-model="form.gpu_ids_str" placeholder="gpu-uuid-1,gpu-uuid-2" />
        </el-form-item>
        <el-form-item :label="t('modelDeployments.hostPort')">
          <el-input-number v-model="form.host_port" :min="0" :max="65535" style="width:100%" />
        </el-form-item>
        <el-form-item :label="t('modelDeployments.servedModelName')">
          <el-input v-model="form.served_model_name" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save" :loading="saving">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Dry Run / Detail Drawer -->
    <el-drawer v-model="detailVisible" :title="t('modelDeployments.dryRunTitle')" size="600px">
      <div v-if="dryRunResult">
        <el-alert :title="dryRunResult.valid ? '✓ Valid' : '✗ ' + t('modelDeployments.validationFailed')" :type="dryRunResult.valid ? 'success' : 'error'" :closable="false" style="margin-bottom:12px" />
        <div v-if="dryRunResult.errors?.length" style="margin-bottom:12px">
          <strong style="color:var(--el-color-danger)">{{ t('common.error') }}:</strong>
          <ul style="margin:4px 0;padding-left:20px"><li v-for="(e,i) in dryRunResult.errors" :key="i">{{ e }}</li></ul>
        </div>
        <div v-if="dryRunResult.warnings?.length" style="margin-bottom:12px">
          <strong style="color:var(--el-color-warning)">Warnings:</strong>
          <ul style="margin:4px 0;padding-left:20px"><li v-for="(w,i) in dryRunResult.warnings" :key="i">{{ w }}</li></ul>
        </div>
        <div v-if="dryRunResult.equivalent_command_preview" style="margin-bottom:16px">
          <h4>{{ t('runTemplates.commandPreview') }}</h4>
          <el-input :model-value="dryRunResult.equivalent_command_preview" type="textarea" :rows="10" readonly style="font-family:monospace;font-size:12px" />
        </div>
      </div>
      <el-empty v-else :description="t('common.loading')" />
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchModelDeployments, createModelDeployment, dryRunDeployment, startDeployment, stopDeployment, type ModelDeployment, type DryRunResponse } from '@/api/modelDeployments'
import { fetchModelArtifacts, type ModelArtifact } from '@/api/modelArtifacts'
import { fetchRuntimeEnvironments, type RuntimeEnvironment } from '@/api/runtimeEnvironments'
import { fetchRunTemplates, type RunTemplate } from '@/api/runTemplates'
import { fetchNodes, type Node } from '@/api/nodes'
import { fetchGPUs } from '@/api/gpus'
import { createModelArtifact } from '@/api/modelArtifacts'
import { createRuntimeEnvironment } from '@/api/runtimeEnvironments'
import { createRunTemplate } from '@/api/runTemplates'
import { formatGB } from '@/utils/format'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { formatDateTime } from '@/utils/format'
import StatusTag from '@/components/StatusTag.vue'

const { t } = useI18n()
// Quick deploy state
const qd = ref({ preset: '', modelPath: '', hostPort: 8002, nodeId: '', gpuId: '', summary: null as Record<string,string>|null })
const nodeGpus = ref<any[]>([])
const quickDeploying = ref(false)

const PRESETS: Record<string, any> = {
  'llama-cpp-nvidia-local': {
    image: 'ghcr.io/ggml-org/llama.cpp:server-cuda13',
    runtime_type: 'docker', backend_type: 'llama_cpp', vendor: 'nvidia', default_port: 8000,
    docker: { image: 'ghcr.io/ggml-org/llama.cpp:server-cuda13', ipc_mode: {enabled:true,value:'host'}, shm_size: {enabled:true,value:'8gb'} },
    args_template: ['-m','${MODEL_PATH}','--host','0.0.0.0','--port','${CONTAINER_PORT}'],
    required_variables: ['MODEL_PATH','CONTAINER_PORT'],
    volume_host_prefix: '', volume_container: '/models',
    isExample: true,
  },
  'llama-cpp-nvidia-custom': {
    image: 'ghcr.io/ggml-org/llama.cpp:server-cuda13',
    runtime_type: 'docker', backend_type: 'llama_cpp', vendor: 'nvidia', default_port: 8000,
    docker: { image: 'ghcr.io/ggml-org/llama.cpp:server-cuda13', ipc_mode: {enabled:true,value:'host'}, shm_size: {enabled:true,value:'8gb'} },
    args_template: ['-m','${MODEL_PATH}','--host','0.0.0.0','--port','${CONTAINER_PORT}'],
    required_variables: ['MODEL_PATH','CONTAINER_PORT'],
    volume_host_prefix: '', volume_container: '/models',
    isExample: false,
  },
  'metax-docker-custom': {
    image: '', runtime_type: 'docker', backend_type: 'custom', vendor: 'metax', default_port: 8000,
    docker: { privileged: {enabled:true,value:true}, ipc_mode: {enabled:true,value:'host'}, uts_mode: {enabled:true,value:'host'}, shm_size: {enabled:true,value:'8gb'}, group_add: {enabled:true,value:'video'}, security_options: {enabled:true,value:'seccomp=unconfined,apparmor=unconfined'}, devices: {enabled:true,value:[{host_path:'/dev/dri',container_path:'/dev/dri'},{host_path:'/dev/mxcd',container_path:'/dev/mxcd'}]} },
    args_template: [],
    required_variables: ['MODEL_PATH'],
    volume_host_prefix: '', volume_container: '/models',
    isExample: false,
  },
  'generic-docker': {
    image: '', runtime_type: 'docker', backend_type: 'custom', vendor: 'custom', default_port: 8000,
    docker: {},
    args_template: [],
    required_variables: [],
    volume_host_prefix: '', volume_container: '',
    isExample: false,
  },
}
function applyPreset(val: string) {
  const p = PRESETS[val]; if (!p) return
  if (p.isExample && !qd.value.modelPath) {
    qd.value.modelPath = ''  // local E2E example: '/home/kzeng/models/...'
  }
  // Show editable fields for custom presets
  qd.value.summary = p.isExample ? {'注意': '这是本地 E2E 示例预设，请根据实际路径修改'} : null
}
async function onNodeChange() { if (qd.value.nodeId) { const gpus = await fetchGPUs({node_id: qd.value.nodeId}); nodeGpus.value = Array.isArray(gpus) ? gpus : [] } }

async function quickDeploy() {
  quickDeploying.value = true; qd.value.summary = null
  try {
    const preset = PRESETS[qd.value.preset]; if (!preset) return
    const modelDir = qd.value.modelPath.substring(0, qd.value.modelPath.lastIndexOf('/'))
    const modelName = qd.value.modelPath.split('/').pop() || 'model'

    // 1. Create artifact
    const art: any = await createModelArtifact({ name: modelName, path: qd.value.modelPath, format: 'gguf', task_type: 'chat', architecture: 'qwen', size_label: '9B', quantization: 'int4' })

    // 2. Create runtime env
    const env: any = await createRuntimeEnvironment({ name: 'qd-' + modelName, runtime_type: preset.runtime_type, backend_type: preset.backend_type, vendor: preset.vendor, default_port: preset.default_port, docker: preset.docker })

    // 3. Create template
    const volHost = preset.volume_host_prefix || modelDir
    const tpl: any = await createRunTemplate({ name: 'qd-' + modelName, runtime_type: preset.runtime_type, vendor: preset.vendor, backend_type: preset.backend_type, required_variables: preset.required_variables, args_template: preset.args_template, volume_mappings: preset.volume_container ? {enabled:true,value:[{host_path:volHost,container_path:preset.volume_container,readonly:true}]} : undefined })

    // 4. Create deployment
    const deploy: any = await createModelDeployment({ name: 'qd-' + modelName, model_artifact_id: art.id, runtime_environment_id: env.id, run_template_id: tpl.id, node_id: qd.value.nodeId, gpu_ids: qd.value.gpuId ? [qd.value.gpuId] : [], host_port: qd.value.hostPort })

    qd.value.summary = { 'Artifact': art.id, 'Runtime': env.id, 'Template': tpl.id, 'Deployment': deploy.id, 'Status': deploy.status }
    ElMessage.success('快速部署完成! Deployment: ' + deploy.id)
    refresh()
  } catch (e: any) { ElMessage.error(e?.message || 'Quick deploy failed') }
  finally { quickDeploying.value = false }
}

const items = ref<ModelDeployment[]>([])
const { loading, refresh } = useAutoRefresh(async () => { items.value = await fetchModelDeployments() }, { intervalMs: 5000 })

const artifacts = ref<ModelArtifact[]>([])
const runtimes = ref<RuntimeEnvironment[]>([])
const templates = ref<RunTemplate[]>([])
const nodes = ref<Node[]>([])

onMounted(async () => {
  artifacts.value = await fetchModelArtifacts()
  runtimes.value = await fetchRuntimeEnvironments()
  templates.value = await fetchRunTemplates()
  nodes.value = await fetchNodes()
})

const dialogVisible = ref(false)
const saving = ref(false)
const detailVisible = ref(false)
const dryRunResult = ref<DryRunResponse | null>(null)

const defaultForm = () => ({
  name: '', model_artifact_id: '', runtime_environment_id: '', run_template_id: '',
  node_id: '', gpu_ids_str: '', host_port: 8001, served_model_name: '',
})
const form = ref(defaultForm())

function resetForm() { form.value = defaultForm() }
function openCreate() { resetForm(); dialogVisible.value = true }
function openDetail(row: ModelDeployment) { openDryRun(row) }

async function openDryRun(row: ModelDeployment) {
  detailVisible.value = true
  dryRunResult.value = null
  try { dryRunResult.value = await dryRunDeployment(row.id) }
  catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
}

function canStart(row: ModelDeployment) {
  return !['running', 'starting', 'pending'].includes(row.status)
}
function canStop(row: ModelDeployment) {
  return ['running'].includes(row.status)
}

async function handleStart(row: ModelDeployment) {
  try {
    const resp = await startDeployment(row.id)
    if (resp.status === 'already_running') ElMessage.info(t('modelDeployments.alreadyRunning'))
    else if (resp.error) ElMessage.error(resp.error)
    else ElMessage.success(t('modelDeployments.startDispatched'))
    refresh()
  } catch (e: any) {
    if (e?.status === 409) ElMessage.warning(t('modelDeployments.alreadyStarting'))
    else if (e?.data?.error?.includes('reserved')) ElMessage.error(t('modelDeployments.leaseConflict'))
    else ElMessage.error(e?.message || t('common.error'))
  }
}

async function handleStop(row: ModelDeployment) {
  try {
    const resp = await stopDeployment(row.id)
    if (resp.status === 'already_stopped') ElMessage.info(t('modelDeployments.alreadyStopped'))
    else ElMessage.success(t('modelDeployments.stopDispatched'))
    refresh()
  } catch (e: any) {
    if (e?.status === 409) ElMessage.warning(t('modelDeployments.alreadyStarting'))
    else ElMessage.error(e?.message || t('common.error'))
  }
}

async function save() {
  saving.value = true
  try {
    const payload: any = {
      ...form.value,
      gpu_ids: form.value.gpu_ids_str.split(',').map((s: string) => s.trim()).filter(Boolean),
    }
    delete (payload as any).gpu_ids_str
    await createModelDeployment(payload)
    ElMessage.success(t('modelDeployments.createSuccess'))
    dialogVisible.value = false
    refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.error')) }
  finally { saving.value = false }
}
</script>
