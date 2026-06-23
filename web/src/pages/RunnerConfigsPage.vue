<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('runnerConfigs.title') }}</h2>
      <el-button type="primary" @click="startWizard">{{ $t('runnerConfigs.newConfig') }}</el-button>
    </div>

    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column prop="name" :label="$t('runnerConfigs.name')" min-width="160" />
      <el-table-column :label="$t('modelLocations.node')" width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.node_label || row.node_id }}</template>
      </el-table-column>
      <el-table-column :label="$t('runnerConfigs.runnerType')" width="100">
        <template #default="{ row }">{{ row.runner_type === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : (row.runner_type || '-') }}</template>
      </el-table-column>
      <el-table-column :label="$t('nodeRuntime.status')" width="100">
        <template #default="{ row }"><el-tag :type="getStatusType(row.status)" size="small">{{ translateStatus(row.status, t) }}</el-tag></template>
      </el-table-column>
      <el-table-column prop="image_ref" :label="$t('nodeRuntime.imageRef')" min-width="220" show-overflow-tooltip />
      <el-table-column prop="last_checked_at" :label="$t('nodeRuntime.lastChecked')" width="180" show-overflow-tooltip />
      <el-table-column :label="$t('common.actions')" width="310">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="warning" @click="checkRow(row)">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button size="small" type="danger" @click="doDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Wizard dialog -->
    <el-dialog v-model="wizardVisible" :title="$t('runnerConfigs.wizardTitle')" width="800px" :close-on-click-modal="false">
      <el-steps :active="step" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('runnerConfigs.selectRunnerType')" />
        <el-step :title="$t('runnerConfigs.selectTemplate')" />
        <el-step :title="$t('runnerConfigs.selectNode')" />
        <el-step :title="$t('runnerConfigs.selectImage')" />
        <el-step :title="$t('runnerConfigs.create')" />
      </el-steps>

      <div v-if="step===0">
        <el-form label-width="140px" style="margin-bottom:12px">
          <el-form-item :label="$t('runnerConfigs.configName')"><el-input v-model="wizConfigName" /></el-form-item>
        </el-form>
        <el-select v-model="wizRunnerType" :placeholder="$t('runnerConfigs.selectRunnerType')" style="width:100%" @change="onWizAutoNext">
          <el-option label="Docker" value="docker" />
        </el-select>
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizRunnerType" @click="step=1">{{ $t('common.next') }}</el-button></div>
      </div>

      <div v-if="step===1">
        <el-select v-model="wizTemplateId" :placeholder="$t('runnerConfigs.selectTemplate')" style="width:100%" filterable @change="onWizTemplateSelected">
          <el-option v-for="t in templates" :key="t.id" :label="`${t.name} (${t.vendor})`" :value="t.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizTemplateId" @click="step=2">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===2">
        <el-select v-model="wizNodeId" :placeholder="$t('runnerConfigs.selectNode')" style="width:100%" filterable @change="onWizAutoNext">
          <el-option v-for="n in nodeItems" :key="n.id" :label="n.label" :value="n.id" />
        </el-select>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=1">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizNodeId" @click="step=wizRunnerType==='docker'?3:4">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===3 && wizRunnerType==='docker'">
        <DockerImagePicker v-if="wizNodeId" :node-id="wizNodeId" @select="onWizardImageSelected" />
        <el-form label-width="130px" style="margin-top:12px">
          <el-form-item :label="$t('dockerImages.selectedImage')"><el-input v-model="wizImageRef" @input="wizImagePresent = false" /></el-form-item>
        </el-form>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=2">{{ $t('common.prev') }}</el-button>
          <span v-if="wizImageRef" class="next-summary">{{ wizImageRef }}</span>
          <el-button type="primary" :disabled="!wizImageRef" @click="step=4">{{ $t('common.next') }}</el-button>
        </div>
      </div>

      <div v-if="step===4">
        <el-form label-width="120px">
          <el-form-item :label="$t('runnerConfigs.template')"><span>{{ wizTemplateId }}</span></el-form-item>
          <el-form-item :label="$t('runnerConfigs.configName')"><el-input v-model="wizConfigName" /></el-form-item>
          <el-form-item :label="$t('runnerConfigs.runnerType')"><span>{{ wizRunnerType === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : wizRunnerType }}</span></el-form-item>
          <el-form-item :label="$t('modelLocations.node')"><span>{{ wizNodeId }}</span></el-form-item>
          <el-form-item v-if="wizImageRef" :label="$t('runnerConfigs.selectImage')"><span>{{ wizImageRef }}</span></el-form-item>
        </el-form>
        <div v-if="wizCheckResult" style="margin-top:8px">
          <el-alert :type="getStatusType(wizCheckResult.status)" :title="translateStatus(wizCheckResult.status, t)" :description="translateStatusReason(wizCheckResult.status_reason, t)" show-icon :closable="false" />
          <div v-if="wizCheckResult.probe_results" style="margin-top:8px">
            <el-collapse>
              <el-collapse-item :title="$t('nodeRuntimeProbe.imageMetadata')" name="level2" v-if="wizCheckResult.probe_results.level2?.inspect_success">
                <el-descriptions :column="2" border size="small">
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.imageId')">{{ (wizCheckResult.probe_results.level2?.image_id || '').slice(7,19) || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.architecture')">{{ wizCheckResult.probe_results.level2?.architecture || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.os')">{{ wizCheckResult.probe_results.level2?.os || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.created')">{{ (wizCheckResult.probe_results.level2?.created || '').slice(0,19) || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.size')">{{ formatBytes(wizCheckResult.probe_results.level2?.size_bytes) }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.entrypoint')">{{ (wizCheckResult.probe_results.level2?.entrypoint || []).join(', ') || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.cmd')">{{ (wizCheckResult.probe_results.level2?.cmd || []).join(', ') || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('nodeRuntimeProbe.exposedPorts')">{{ Object.keys(wizCheckResult.probe_results.level2?.exposed_ports || {}).join(', ') || '-' }}</el-descriptions-item>
                  <el-descriptions-item :span="2" :label="$t('nodeRuntimeProbe.repotags')">{{ (wizCheckResult.probe_results.level2?.repotags || []).join(', ') || '-' }}</el-descriptions-item>
                </el-descriptions>
              </el-collapse-item>
              <el-collapse-item :title="$t('nodeRuntimeProbe.backendMatch')" name="level3" v-if="wizCheckResult.probe_results.level3">
                <p>{{ wizCheckResult.probe_results.level3.match_detail || $t('nodeRuntimeProbe.notChecked') }}</p>
              </el-collapse-item>
              <el-collapse-item :title="$t('nodeRuntimeProbe.versionProbe')" name="level4" v-if="wizCheckResult.probe_results.level4">
                <p v-if="wizCheckResult.probe_results.level4.version_probed">{{ wizCheckResult.probe_results.level4.version_string }}</p>
                <p v-else>{{ $t('nodeRuntimeProbe.notProbed') }}: {{ wizCheckResult.probe_results.level4.probe_error || $t('nodeRuntimeProbe.notProbed') }}</p>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>
        <div style="margin-top:12px;text-align:right">
          <el-button @click="step=wizRunnerType==='docker'?3:2">{{ $t('common.prev') }}</el-button>
          <el-button @click="doCheck" :loading="checking">{{ $t('runnerConfigs.check') }}</el-button>
          <el-button type="primary" :disabled="!wizCheckResult || (wizCheckResult.status !== 'ready' && wizCheckResult.status !== 'ready_with_warnings')" @click="doCreateConfig" :loading="saving">{{ $t('runnerConfigs.create') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Detail drawer -->
    <el-drawer v-model="detailVisible" :title="$t('common.detail')" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runnerConfigs.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelLocations.node')">{{ selected.node_label || selected.node_id }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.runnerType')">{{ selected.runner_type === 'docker' ? $t('runnerConfigs.runnerTypeDocker') : (selected.runner_type || '-') }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.status')">
            <el-tag :type="getStatusType(selected.status)" size="small">{{ translateStatus(selected.status, t) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.imageRef')">{{ selected.image_ref || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.template')">{{ selected.template_name || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nodeRuntime.statusReason')" :span="2">{{ translateStatusReason(selected.status_reason, t) }}</el-descriptions-item>
        </el-descriptions>
        <h4 style="margin-top:16px">{{ $t('runnerConfigs.sectionImageCommand') }}</h4>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('nodeRuntime.imageRef')">{{ runParamSummary(selected).image || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.entrypoint')">{{ runParamSummary(selected).entrypoint || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.command')" :span="2">{{ runParamSummary(selected).command || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runnerConfigs.args')" :span="2">{{ runParamSummary(selected).args || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top:16px">{{ $t('runnerConfigs.sectionEnv') }}</h4>
        <el-table :data="runParamSummary(selected).envRows" stripe size="small" empty-text="-">
          <el-table-column prop="key" :label="$t('runnerConfigs.envKey')" width="260" />
          <el-table-column prop="value" :label="$t('runnerConfigs.envValue')" show-overflow-tooltip />
        </el-table>

        <h4 style="margin-top:16px">{{ $t('runnerConfigs.sectionVolumesPorts') }}</h4>
        <el-table :data="runParamSummary(selected).volumeRows" stripe size="small" empty-text="-">
          <el-table-column prop="host" :label="$t('runnerConfigs.hostPath')" min-width="220" show-overflow-tooltip />
          <el-table-column prop="container" :label="$t('runnerConfigs.containerPath')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="readonly" :label="$t('runnerConfigs.readonly')" width="90" />
        </el-table>
        <el-table :data="runParamSummary(selected).portRows" stripe size="small" empty-text="-" style="margin-top:8px">
          <el-table-column prop="host" :label="$t('deployments.hostPort')" width="160" />
          <el-table-column prop="container" :label="$t('deployments.containerPort')" width="180" />
          <el-table-column prop="protocol" :label="$t('runnerConfigs.protocol')" width="120" />
        </el-table>

        <h4 style="margin-top:16px">{{ $t('runnerConfigs.sectionDevicesSecurity') }}</h4>
        <el-alert v-if="runParamSummary(selected).riskText" type="warning" :title="runParamSummary(selected).riskText" show-icon :closable="false" style="margin-bottom:8px" />
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('runtimes.devices')">{{ runParamSummary(selected).devices || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.groupAdd')">{{ runParamSummary(selected).groupAdd || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.privileged')">{{ runParamSummary(selected).privileged || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.ipcMode')">{{ runParamSummary(selected).ipc || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.securityOpt')">{{ runParamSummary(selected).securityOpt || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.shmSize')">{{ runParamSummary(selected).shmSize || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('runtimes.ulimits')" :span="2">{{ runParamSummary(selected).ulimits || '-' }}</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top:16px">{{ $t('runnerConfigs.sectionHealthPreview') }}</h4>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item :label="$t('runnerConfigs.nbrTemplatePreview')">{{ runParamSummary(selected).preview || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div style="margin-top:8px">
          <JsonViewer :value="runParamSummary(selected).healthObj" title="Health Check Config" max-height="300px" />
        </div>
        <el-collapse v-if="selected?.probe_results_json && typeof selected.probe_results_json === 'object' && Object.keys(selected.probe_results_json).length > 0" style="margin-top:12px">
          <el-collapse-item :title="$t('nodeRuntimeProbe.imageMetadata')" name="level2" v-if="selected.probe_results_json.level2?.inspect_success">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item :label="$t('nodeRuntimeProbe.imageId')">{{ (selected.probe_results_json.level2?.image_id || '').slice(7,19) || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.architecture')">{{ selected.probe_results_json.level2?.architecture || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.os')">{{ selected.probe_results_json.level2?.os || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.created')">{{ (selected.probe_results_json.level2?.created || '').slice(0,19) || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.size')">{{ formatBytes(selected.probe_results_json.level2?.size_bytes) }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.entrypoint')">{{ (selected.probe_results_json.level2?.entrypoint || []).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.cmd')">{{ (selected.probe_results_json.level2?.cmd || []).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('nodeRuntimeProbe.exposedPorts')">{{ Object.keys(selected.probe_results_json.level2?.exposed_ports || {}).join(', ') || '-' }}</el-descriptions-item>
              <el-descriptions-item :span="2" :label="$t('nodeRuntimeProbe.repotags')">{{ (selected.probe_results_json.level2?.repotags || []).join(', ') || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
          <el-collapse-item :title="$t('nodeRuntimeProbe.backendMatch')" name="level3" v-if="selected.probe_results_json.level3">
            <p>{{ selected.probe_results_json.level3.match_detail || $t('nodeRuntimeProbe.notChecked') }}</p>
          </el-collapse-item>
          <el-collapse-item :title="$t('nodeRuntimeProbe.versionProbe')" name="level4" v-if="selected.probe_results_json.level4">
            <p v-if="selected.probe_results_json.level4.version_probed">{{ selected.probe_results_json.level4.version_string }}</p>
            <p v-else>{{ $t('nodeRuntimeProbe.notProbed') }}: {{ selected.probe_results_json.level4.probe_error || $t('nodeRuntimeProbe.notProbed') }}</p>
          </el-collapse-item>
          <el-collapse-item :title="$t('nodeRuntimeProbe.backendMatch')" name="level3" v-if="selected.probe_results_json.level3">
            <p>{{ selected.probe_results_json.level3.match_detail || $t('nodeRuntimeProbe.notChecked') }}</p>
          </el-collapse-item>
          <el-collapse-item :title="$t('nodeRuntimeProbe.versionProbe')" name="level4" v-if="selected.probe_results_json.level4">
            <p v-if="selected.probe_results_json.level4.version_probed">{{ selected.probe_results_json.level4.version_string }}</p>
            <p v-else>{{ $t('nodeRuntimeProbe.notProbed') }}: {{ selected.probe_results_json.level4.probe_error || $t('nodeRuntimeProbe.notProbed') }}</p>
          </el-collapse-item>
        </el-collapse>
        <!-- Diagnostic notices -->
        <el-alert v-if="isShellWrapper(selected.probe_results_json?.level2?.entrypoint)" type="info" :title="$t('nodeRuntimeProbe.shellWrapper')" show-icon :closable="false" style="margin-top:8px" />
        <el-alert v-if="isVendorImage(selected.probe_results_json?.level3)" type="warning" :title="$t('nodeRuntimeProbe.vendorImage')" show-icon :closable="false" style="margin-top:8px" />
        <el-alert v-if="isBlockingError(selected.status)" type="error" :title="translateStatus(selected.status, t)" :description="translateStatusReason(selected.status_reason, t)" show-icon :closable="false" style="margin-top:8px" />
        <!-- Advanced diagnostic JSON -->
        <el-collapse v-if="selected?.config_snapshot_json && Object.keys(selected.config_snapshot_json).length > 0" style="margin-top:12px">
          <el-collapse-item :title="$t('runnerConfigs.advancedJson')" name="runParams">
            <JsonViewer :value="selected.config_snapshot_json" title="Config Snapshot" max-height="500px" :allow-download="true" :searchable="true" />
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>

    <el-dialog v-model="editVisible" :title="$t('runnerConfigs.editConfig')" width="760px">
      <el-alert :title="$t('runnerConfigs.editAffectsNextStart')" type="warning" show-icon :closable="false" style="margin-bottom:12px" />
      <el-form label-position="top">
        <el-form-item :label="$t('runnerConfigs.configName')"><el-input v-model="editConfigName" /></el-form-item>
        <el-form-item :label="$t('nodeRuntime.imageRef')"><el-input v-model="editImageRef" /></el-form-item>
        <h4>{{ $t('runnerConfigs.sectionImageCommand') }}</h4>
        <el-form-item :label="$t('runnerConfigs.args')"><el-input v-model="editArgsText" type="textarea" :rows="3" :placeholder="$t('runnerConfigs.lineSeparated')" /></el-form-item>
        <h4>{{ $t('runnerConfigs.sectionEnv') }}</h4>
        <el-form-item :label="$t('runtimes.env')"><el-input v-model="editEnvText" type="textarea" :rows="4" :placeholder="$t('runnerConfigs.keyValueLines')" /></el-form-item>
        <h4>{{ $t('runnerConfigs.sectionVolumesPorts') }}</h4>
        <el-form-item :label="$t('runtimes.extraMounts')"><el-input v-model="editVolumesText" type="textarea" :rows="3" :placeholder="$t('runnerConfigs.volumeLines')" /></el-form-item>
        <el-form-item :label="$t('runnerConfigs.ports')"><el-input v-model="editPortsText" type="textarea" :rows="3" :placeholder="$t('runnerConfigs.portLines')" /></el-form-item>
        <h4>{{ $t('runnerConfigs.sectionDevicesSecurity') }}</h4>
        <el-alert :title="$t('runnerConfigs.highRiskWarning')" type="warning" show-icon :closable="false" style="margin-bottom:8px" />
        <el-form-item :label="$t('runtimes.devices')"><el-input v-model="editDevicesText" type="textarea" :rows="3" :placeholder="$t('runnerConfigs.volumeLines')" /></el-form-item>
        <el-form-item :label="$t('runtimes.groupAdd')"><el-input v-model="editGroupAddText" type="textarea" :rows="2" :placeholder="$t('runnerConfigs.lineSeparated')" /></el-form-item>
        <el-form-item :label="$t('runtimes.securityOpt')"><el-input v-model="editSecurityOptText" type="textarea" :rows="2" :placeholder="$t('runnerConfigs.lineSeparated')" /></el-form-item>
        <el-form-item :label="$t('runtimes.privileged')"><el-switch v-model="editPrivileged" /></el-form-item>
        <el-form-item :label="$t('runtimes.ipcMode')"><el-input v-model="editIpcMode" /></el-form-item>
        <el-form-item :label="$t('runtimes.shmSize')"><el-input v-model="editShmSize" /></el-form-item>
        <el-form-item :label="$t('runtimes.ulimits')"><el-input v-model="editUlimitsText" type="textarea" :rows="2" :placeholder="$t('runnerConfigs.keyValueLines')" /></el-form-item>
        <h4>{{ $t('runnerConfigs.sectionHealthPreview') }}</h4>
        <el-form-item :label="$t('backends.healthCheck')">
          <HealthCheckEditor v-model="editHealthModel" />
        </el-form-item>
        <h4>{{ $t('runtimes.structuredParameters') }}</h4>
        <RuntimeParameterEditor v-model="editParameterModel" />
        <el-collapse>
          <el-collapse-item :title="$t('runnerConfigs.advancedJson')">
            <el-form-item :label="$t('runnerConfigs.snapshotJson')"><el-input v-model="editSnapshotText" type="textarea" :rows="8" /></el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="doEdit" :loading="saving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import { listRuntimes } from '@/api/runtimes'
import DockerImagePicker from '@/components/DockerImagePicker.vue'
import JsonViewer from '@/components/common/JsonViewer.vue'
import HealthCheckEditor from '@/components/common/HealthCheckEditor.vue'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'
import { getStatusType, translateStatus, translateStatusReason } from '@/utils/status'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'
const { loadNodes, nodes: nodeItems, nodeLabel } = useNodeLabels()
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

const loading = ref(false); const saving = ref(false); const checking = ref(false)
const items = ref<any[]>([]); const templates = ref<any[]>([]); const selected = ref<any>(null); const detailVisible = ref(false)
const editVisible = ref(false); const editConfigName = ref(''); const editImageRef = ref(''); const editSnapshotText = ref('{}')
const editArgsText = ref(''); const editEnvText = ref(''); const editVolumesText = ref(''); const editPortsText = ref('')
const editDevicesText = ref(''); const editGroupAddText = ref(''); const editSecurityOptText = ref('')
const editPrivileged = ref(false); const editIpcMode = ref(''); const editShmSize = ref(''); const editUlimitsText = ref(''); const editHealthText = ref('{}')
const editHealthModel = ref<Record<string, unknown>>({})
const editParameterModel = ref({ docker_json: {}, args_override_json: [], default_env_json: {}, parameter_values_json: [] })

// Wizard
const wizardVisible = ref(false); const step = ref(0)
const wizTemplateId = ref(''); const wizRunnerType = ref('docker')
const wizNodeId = ref(''); const wizImageRef = ref(''); const wizImagePresent = ref(false)
const wizConfigName = ref(''); const wizCheckResult = ref<any>(null)

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(step, () => { step.value++ })

function isShellWrapper(entrypoint: any): boolean {
  if (!entrypoint || !Array.isArray(entrypoint) || entrypoint.length === 0) return false
  const ep = entrypoint[0]
  if (!ep || typeof ep !== 'string') return false
  const shells = ['bash', 'sh', '/bin/bash', '/bin/sh', '/usr/bin/bash', '/usr/bin/sh',
    'python', 'python3', '/usr/bin/python', '/usr/bin/python3', '/usr/local/bin/python', '/usr/local/bin/python3']
  return shells.includes(ep) || shells.some(s => ep.endsWith('/' + s))
}
function isVendorImage(level3: any): boolean {
  if (!level3) return false
  return level3.backend_match_status === 'declared_match_unverified'
}
function isBlockingError(status: string): boolean {
  return status === 'missing_image' || status === 'inspect_failed' ||
    status === 'docker_error' || status === 'agent_unreachable'
}
function formatBytes(bytes: any): string {
  if (bytes == null || bytes === 0) return '-'
  const n = Number(bytes)
  if (isNaN(n)) return '-'
  if (n < 1024) return n + ' ' + t('nodeRuntimeProbe.bytes')
  if (n < 1048576) return (n / 1024).toFixed(1) + ' ' + t('nodeRuntimeProbe.kb')
  if (n < 1073741824) return (n / 1048576).toFixed(1) + ' ' + t('nodeRuntimeProbe.mb')
  return (n / 1073741824).toFixed(2) + ' ' + t('nodeRuntimeProbe.gb')
}

function asObject(value: any): Record<string, any> {
  if (!value) return {}
  if (typeof value === 'object' && !Array.isArray(value)) return value
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {}
    } catch { return {} }
  }
  return {}
}

function asArray(value: any): any[] {
  if (Array.isArray(value)) return value
  if (value == null || value === '') return []
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (Array.isArray(parsed)) return parsed
    } catch {
      return value.split('\n').map((v: string) => v.trim()).filter(Boolean)
    }
  }
  return [value]
}

function joinList(value: any): string {
  return asArray(value).map(formatRuntimeValue).filter(Boolean).join('\n')
}

function formatRuntimeValue(value: any): string {
  if (value == null || value === '') return ''
  if (typeof value === 'string') return value
  if (typeof value === 'object') {
    if (value.host_path || value.container_path) {
      return `${value.host_path || ''}:${value.container_path || value.host_path || ''}${value.readonly ? ':ro' : value.permissions ? ':' + value.permissions : ''}`
    }
    if (value.host_port || value.container_port) {
      return `${value.host_port || ''}:${value.container_port || ''}/${value.protocol || 'tcp'}`
    }
    return JSON.stringify(value)
  }
  return String(value)
}

function envRows(env: any): { key: string, value: string }[] {
  const obj = asObject(env)
  return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }))
}

function parseLines(value: string): string[] {
  return Array.from(new Set((value || '').split('\n').map((v: string) => v.trim()).filter(Boolean)))
}

function parseKeyValueLines(value: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const line of parseLines(value)) {
    const idx = line.indexOf('=')
    if (idx > 0) out[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  }
  return out
}

function parseMountLines(value: string): any[] {
  return parseLines(value).map((line: string) => {
    const parts = line.split(':')
    return { host_path: parts[0] || '', container_path: parts[1] || parts[0] || '', readonly: parts[2] === 'ro', permissions: parts[2] && parts[2] !== 'ro' ? parts[2] : undefined }
  }).filter((m: any) => m.host_path || m.container_path)
}

function parsePortLines(value: string): any[] {
  return parseLines(value).map((line: string) => {
    const [left, protoRaw = 'tcp'] = line.split('/')
    const [host, container = host] = left.split(':')
    return { host_port: Number(host) || 0, container_port: Number(container) || 0, protocol: protoRaw || 'tcp' }
  }).filter((p: any) => p.host_port || p.container_port)
}

function runParamSummary(row: any) {
  const snapshot = asObject(row?.config_snapshot_json)
  const docker = asObject(snapshot.docker_json || snapshot.docker || snapshot)
  const image = snapshot.image_name || snapshot.image || row?.image_ref || ''
  const entrypoint = joinList(snapshot.entrypoint_override_json || snapshot.entrypoint || docker.entrypoint)
  const command = joinList(snapshot.command_override_json || snapshot.command || docker.command)
  const args = joinList(snapshot.args_override_json || snapshot.args || snapshot.extra_args || docker.args)
  const env = snapshot.default_env_json || snapshot.default_env || snapshot.env || docker.default_env || docker.env
  const volumes = snapshot.extra_mounts || snapshot.volumes_json || snapshot.volumes || snapshot.mounts || docker.extra_mounts || docker.volumes
  const ports = snapshot.ports_json || snapshot.ports || docker.ports
  const devices = joinList(docker.devices || snapshot.devices)
  const groupAdd = joinList(docker.group_add || snapshot.group_add)
  const securityOpt = joinList(docker.security_options || docker.security_opt || snapshot.security_opt)
  const privileged = docker.privileged === true || snapshot.privileged === true ? t('common.yes') : ''
  const ipc = docker.ipc_mode || snapshot.ipc || snapshot.ipc_mode || ''
  const shmSize = docker.shm_size || snapshot.shm_size || ''
  const ulimits = typeof docker.ulimits === 'object' ? JSON.stringify(docker.ulimits) : joinList(docker.ulimits || snapshot.ulimits)
  const health = JSON.stringify(snapshot.health_check_override_json || snapshot.health_check_json || snapshot.health_check || {}, null, 2)
  const healthObj = snapshot.health_check_override_json || snapshot.health_check_json || snapshot.health_check || {}
  const volumeRows = asArray(volumes).map((v) => {
    if (typeof v === 'string') {
      const parts = v.split(':')
      return { host: parts[0] || '', container: parts[1] || parts[0] || '', readonly: parts[2] === 'ro' ? t('common.yes') : t('common.no') }
    }
    return { host: v.host_path || v.host || '', container: v.container_path || v.container || '', readonly: v.readonly ? t('common.yes') : t('common.no') }
  })
  const portRows = asArray(ports).map((p) => typeof p === 'string'
    ? { host: p.split(':')[0] || '', container: (p.split(':')[1] || '').split('/')[0], protocol: p.includes('/') ? p.split('/')[1] : 'tcp' }
    : { host: p.host_port || p.host || '', container: p.container_port || p.container || '', protocol: p.protocol || 'tcp' })
  const riskText = (docker.privileged || ipc === 'host' || securityOpt) ? t('runnerConfigs.highRiskWarning') : ''
  const preview = image ? ['docker run -d', docker.privileged ? '--privileged' : '', ipc ? `--ipc ${ipc}` : '', shmSize ? `--shm-size ${shmSize}` : '', image, args].filter(Boolean).join(' ') : ''
  return { image, entrypoint, command, args, envRows: envRows(env), volumeRows, portRows, devices, groupAdd, privileged, ipc, securityOpt, shmSize, ulimits, health, healthObj, riskText, preview }
}

onMounted(async () => { await loadRefs(); await refresh() })

async function refresh() {
  loading.value = true
  try {
    // Collect NodeBackendRuntime records from all nodes
    const nbrList: any[] = []
    for (const n of nodeItems.value) {
      try {
        const nbrs = await apiClient.get(`/nodes/${n.id}/backend-runtimes`)
        if (Array.isArray(nbrs)) {
          for (const nbr of nbrs) {
            nbrList.push({ ...nbr, _node_label: n.label, _node_id: n.id })
          }
        }
      } catch {}
    }
    items.value = nbrList.map((nbr: any) => ({
      id: nbr.id,
      name: nbr.display_name || nbr.name || nbr.backend_runtime?.display_name || nbr.backend_runtime?.name || nbr.backend_runtime_id,
      template_name: nbr.backend_runtime?.name || '-',
      runner_type: nbr.runner_type || 'docker',
      node_count: 1,
      ready_count: nbr.status === 'ready' ? 1 : 0,
      status: nbr.status,
      node_id: nbr._node_id,
        node_label: nbr._node_label,
        image_ref: nbr.image_ref,
        image_present: nbr.image_present,
        last_checked_at: nbr.last_checked_at,
        status_reason: nbr.status_reason,
        config_snapshot_json: nbr.config_snapshot_json || {},
        probe_results_json: nbr.probe_results_json || {},
        backend_runtime_id: nbr.backend_runtime_id,
    }))
  } catch {}
  loading.value = false
}

async function loadRefs() {
  try { templates.value = await listRuntimes() } catch { templates.value = [] }
  loadNodes()
}

function startWizard() { wizardVisible.value = true; step.value = 0; wizTemplateId.value = ''; wizRunnerType.value = 'docker'; wizNodeId.value = ''; wizImageRef.value = ''; wizImagePresent.value = false; wizConfigName.value = ''; wizCheckResult.value = null; loadRefs() }

function onWizTemplateSelected(templateId: string) {
  const template = templates.value.find((t: any) => t.id === templateId)
  if (!template) return
  // Only auto-generate name if user hasn't entered a custom one
  if (!wizConfigName.value || wizConfigName.value.trim() === '') {
    const suffix = t('runnerConfigs.customSuffix')
    const baseName = `${template.name}${suffix}`
    // Auto-append number if name conflicts with existing configs
    const existingNames = new Set(items.value.map((c: any) => c.name))
    let candidate = baseName
    let counter = 2
    while (existingNames.has(candidate)) {
      candidate = `${baseName} ${counter}`
      counter++
    }
    wizConfigName.value = candidate
  }
  // Auto-advance: this step has only one select control
  step.value = 2
}

async function doCheck() {
  checking.value = true
  try {
    // First enable (create/update NBR), then trigger server-side check.
    const nbr: any = await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: wizTemplateId.value, display_name: wizConfigName.value, image_ref: wizImageRef.value || '' })
    const nbrId = nbr?.id
    if (!nbrId) { wizCheckResult.value = { status: 'unknown', status_reason: 'failed to create node runtime config' }; checking.value = false; return }
    wizCheckResult.value = await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/${nbrId}/check-request`, {})
  } catch (e: any) { wizCheckResult.value = { status: 'unknown', status_reason: e?.message || 'check failed' } }
  checking.value = false
}

function onWizardImageSelected(img: any) {
  wizImageRef.value = img.image_ref || ''
  wizCheckResult.value = null
}

async function doCreateConfig() {
  saving.value = true
  try {
    // If check was already done and succeeded, the NBR already exists with ready status.
    // Don't re-enable, which would reset the status to 'needs_check'.
    const checked = wizCheckResult.value
    const alreadyReady = checked && (checked.status === 'ready' || checked.status === 'ready_with_warnings')
    if (!alreadyReady) {
      // Enable the selected template on the selected node (creates NodeBackendRuntime only, no BackendRuntime clone)
      await apiClient.post(`/nodes/${wizNodeId.value}/backend-runtimes/enable`, { backend_runtime_id: wizTemplateId.value, display_name: wizConfigName.value, image_ref: wizImageRef.value })
    }
    ElMessage.success(t('runnerConfigs.created')); wizardVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function showDetail(row: any) {
  selected.value = row
  detailVisible.value = true
}

function showEdit(row: any) {
  selected.value = row
  const summary = runParamSummary(row)
  editConfigName.value = row.name || ''
  editImageRef.value = row.image_ref || ''
  editSnapshotText.value = JSON.stringify(row.config_snapshot_json || {}, null, 2)
  editArgsText.value = summary.args || ''
  editEnvText.value = summary.envRows.map((r) => `${r.key}=${r.value}`).join('\n')
  editVolumesText.value = summary.volumeRows.map((r) => `${r.host}:${r.container}${r.readonly === t('common.yes') ? ':ro' : ''}`).join('\n')
  editPortsText.value = summary.portRows.map((r) => `${r.host}:${r.container}/${r.protocol}`).join('\n')
  editDevicesText.value = joinList(asObject(row.config_snapshot_json || {}).docker_json?.devices || asObject(row.config_snapshot_json || {}).devices)
  editGroupAddText.value = summary.groupAdd.split(', ').join('\n')
  editSecurityOptText.value = summary.securityOpt.split(', ').join('\n')
  editPrivileged.value = summary.privileged === t('common.yes')
  editIpcMode.value = summary.ipc
  editShmSize.value = summary.shmSize
  editUlimitsText.value = (() => {
    try { return Object.entries(JSON.parse(summary.ulimits || '{}')).map(([k, v]) => `${k}=${v}`).join('\n') } catch { return summary.ulimits }
  })()
  editHealthText.value = summary.health || '{}'
  try { editHealthModel.value = JSON.parse(summary.health || '{}') } catch { editHealthModel.value = {} }
  editVisible.value = true
}

async function doEdit() {
  if (!selected.value) return
  saving.value = true
  try {
    let snapshot: any = {}
    try { snapshot = JSON.parse(editSnapshotText.value || '{}') } catch { ElMessage.error(t('runnerConfigs.invalidJson')); saving.value = false; return }
    snapshot.image_name = editImageRef.value
    snapshot.args_override_json = parseLines(editArgsText.value)
    snapshot.default_env_json = parseKeyValueLines(editEnvText.value)
    snapshot.extra_mounts = parseMountLines(editVolumesText.value)
    snapshot.ports_json = parsePortLines(editPortsText.value)
    snapshot.docker_json = asObject(snapshot.docker_json)
    snapshot.docker_json.devices = parseMountLines(editDevicesText.value)
    snapshot.docker_json.group_add = parseLines(editGroupAddText.value)
    snapshot.docker_json.security_options = parseLines(editSecurityOptText.value)
    snapshot.docker_json.privileged = editPrivileged.value
    if (editIpcMode.value) snapshot.docker_json.ipc_mode = editIpcMode.value
    if (editShmSize.value) snapshot.docker_json.shm_size = editShmSize.value
    snapshot.docker_json.ulimits = parseKeyValueLines(editUlimitsText.value)
    snapshot.health_check_override_json = editHealthModel.value
    await apiClient.patch(`/nodes/${selected.value.node_id}/backend-runtimes/${selected.value.id}`, { display_name: editConfigName.value, image_ref: editImageRef.value, config_snapshot_json: snapshot })
    ElMessage.success(t('runnerConfigs.savedNeedsCheck'))
    editVisible.value = false
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function checkRow(row: any) {
  checking.value = true
  try {
    const result = await apiClient.post(`/nodes/${row.node_id}/backend-runtimes/${row.id}/check-request`, {})
    ElMessage.success(`${translateStatus(result.status, t)}: ${translateStatusReason(result.status_reason, t)}`)
    await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  checking.value = false
}

async function doDelete(row: any) {
  try {
    await ElMessageBox.confirm(t('runnerConfigs.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
    // Delete the NodeBackendRuntime record (node-level config only; template is preserved)
    await apiClient.delete(`/nodes/${row.node_id}/backend-runtimes/${row.id}`)
    ElMessage.success(t('runnerConfigs.deleted')); await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
.next-summary { color: var(--el-text-color-secondary); margin-right: 12px; font-size: 12px; }
</style>
