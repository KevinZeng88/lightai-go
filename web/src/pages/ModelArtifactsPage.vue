<template>
  <div class="page-container">
    <div class="page-header">
      <h2>{{ $t('artifacts.title') }}</h2>
      <div>
        <el-button type="primary" @click="startWizard">{{ $t('modelWizard.title') }}</el-button>
      </div>
    </div>
    <el-table :data="items" v-loading="loading" stripe>
      <el-table-column :label="$t('artifacts.displayName')" min-width="180">
        <template #default="{ row }">{{ row.display_name || row.name }}</template>
      </el-table-column>
      <el-table-column prop="name" :label="$t('artifacts.name')" min-width="150" />
      <el-table-column prop="format" :label="$t('artifacts.format')" width="100" />
      <el-table-column prop="size_label" :label="$t('artifacts.size')" width="80" />
      <el-table-column :label="$t('artifacts.capabilities')" min-width="180">
        <template #default="{ row }">
          <el-tag v-for="cap in capabilitiesFor(row).slice(0, 3)" :key="cap.id" size="small" class="cap-tag">
            {{ capabilityText(cap) }}
          </el-tag>
          <span v-if="capabilitiesFor(row).length === 0">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="path" :label="$t('artifacts.path')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="$t('common.actions')" width="280">
        <template #default="{ row }">
          <el-button size="small" @click="showDetail(row)">{{ $t('common.detail') }}</el-button>
          <el-button size="small" @click="showEdit(row)">{{ $t('common.edit') }}</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Edit/Create Dialog -->
    <el-dialog v-model="dialogVisible" :title="editingId ? $t('common.edit') : $t('common.create')" width="700px">
      <el-form :model="form" label-width="140px">
        <!-- Editable basic info -->
        <el-divider content-position="left">{{ $t('artifacts.editSectionBasic') }}</el-divider>
        <el-form-item :label="$t('artifacts.name')">
          <el-input v-model="form.name" :disabled="!!editingId" />
          <el-tag v-if="editingId" size="small" type="info" style="margin-left:8px">{{ $t('common.readonly') }}</el-tag>
        </el-form-item>
        <el-form-item :label="$t('artifacts.displayName')"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item :label="$t('artifacts.path')"><el-input v-model="form.path" /></el-form-item>
        <el-form-item :label="$t('artifacts.format')"><el-select v-model="form.format" filterable allow-create style="width:100%"><el-option v-for="o in formatOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
        <el-form-item :label="$t('artifacts.quantization')"><el-select v-model="form.quantization" filterable allow-create style="width:100%"><el-option v-for="o in quantOptions" :key="o" :label="o" :value="o" /></el-select></el-form-item>
        <el-form-item :label="$t('artifacts.taskType')">
          <el-select v-model="editTaskType" style="width:100%">
            <el-option v-for="o in TASK_TYPE_OPTIONS" :key="o.value" :label="$t(o.labelKey)" :value="o.value" />
          </el-select>
        </el-form-item>

        <!-- Capabilities (editable) -->
        <el-divider content-position="left">{{ $t('artifacts.capabilitySection') }}</el-divider>
        <el-form-item :label="$t('artifacts.capabilities')">
          <el-checkbox-group v-model="editCapabilities">
            <el-checkbox v-for="c in CAPABILITY_OPTIONS" :key="c.value" :label="c.value" :value="c.value">{{ $t(c.labelKey) }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="$t('artifacts.defaultTestMode')">
          <el-select v-model="editDefaultTestMode" style="width:220px">
            <el-option v-for="o in TEST_MODE_OPTIONS" :key="o.value" :label="$t(o.labelKey)" :value="o.value" />
          </el-select>
        </el-form-item>

        <!-- Scan facts (read-only, only in edit mode) -->
        <template v-if="editingId">
          <el-divider content-position="left">{{ $t('artifacts.editSectionScanFacts') }}</el-divider>
          <el-form-item :label="$t('artifacts.size')"><el-input :model-value="form.size_label || '-'" disabled /></el-form-item>
          <el-form-item :label="$t('artifacts.architecture')"><el-input :model-value="form.architecture !== 'custom' ? form.architecture : $t('artifacts.notIdentified')" disabled /></el-form-item>
          <el-form-item :label="$t('artifacts.contextLength')"><el-input :model-value="form.default_context_length || '-'" disabled /></el-form-item>
          <el-form-item :label="$t('artifacts.taskType')"><el-input :model-value="form.task_type || '-'" disabled /></el-form-item>
        </template>

        <!-- Model Facts and Hints (model-level metadata, NOT Docker/runtime configuration) -->
        <el-divider content-position="left">{{ $t('artifacts.parameterDefaults') }}</el-divider>
        <el-alert type="info" :closable="false" style="margin-bottom:8px">
          {{ $t('artifacts.parameterDefaultsHint') }}
        </el-alert>
        <el-form-item :label="$t('artifacts.servingParams')">
          <el-input v-model="parameterDefaultsText" type="textarea" :rows="4" :placeholder="$t('artifacts.parameterDefaultsPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button><el-button type="primary" @click="doSave" :loading="saving">{{ $t('common.save') }}</el-button></template>
    </el-dialog>

    <!-- Detail Dialog with Locations -->
    <el-drawer v-model="detailVisible" :title="$t('artifacts.title')" size="65%">
      <template v-if="selected">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('artifacts.name')">{{ selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.displayName')">{{ selected.display_name || selected.name }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.format')">{{ selected.format }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.taskType')">
            <el-tag size="small" type="primary">{{ taskTypeText(selected.task_type || 'chat') }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.path')">{{ selected.path }}</el-descriptions-item>
          <el-descriptions-item v-if="detailPathType" :label="$t('artifacts.pathType')">{{ detailPathType }}</el-descriptions-item>
          <el-descriptions-item v-if="detailFileSize" :label="$t('artifacts.fileSize')">{{ detailFileSize }}</el-descriptions-item>
          <el-descriptions-item v-if="selected.size_label" :label="$t('artifacts.size')">{{ selected.size_label }}</el-descriptions-item>
          <el-descriptions-item v-if="detailParamCount" :label="$t('artifacts.paramCount')">{{ detailParamCount }}</el-descriptions-item>
          <el-descriptions-item v-if="detailCtxLen" :label="$t('artifacts.contextLength')">{{ detailCtxLen }}</el-descriptions-item>
          <el-descriptions-item v-if="selected.quantization && selected.quantization !== 'unknown'" :label="$t('artifacts.quantization')">{{ selected.quantization }}</el-descriptions-item>
          <el-descriptions-item v-if="selected.architecture && selected.architecture !== 'custom'" :label="$t('artifacts.architecture')">{{ selected.architecture }}</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top:12px">{{ $t('artifacts.capabilitySection') }}</h4>
        <el-alert
          v-if="!hasPersistedCapabilities"
          type="info"
          :closable="false"
          style="margin-bottom:8px"
          :title="$t('artifacts.capabilityReadonlyHint')"
        />
        <el-tag
          v-if="hasPersistedCapabilities"
          type="success"
          size="small"
          style="margin-bottom:8px"
        >{{ $t('artifacts.capabilityPersisted') }}</el-tag>
        <el-table :data="detailCapabilities" stripe size="small">
          <el-table-column :label="$t('artifacts.capability')" width="160">
            <template #default="{ row }">
              <el-tag size="small">{{ capabilityText(row) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('artifacts.capabilitySource')" width="140">
            <template #default="{ row }">{{ capabilitySourceText(row.source) }}</template>
          </el-table-column>
          <el-table-column :label="$t('artifacts.capabilityConfidence')" width="120">
            <template #default="{ row }">{{ capabilityConfidenceText(row.confidence) }}</template>
          </el-table-column>
          <el-table-column prop="reason" :label="$t('artifacts.capabilityReason')" min-width="180" />
        </el-table>
        <el-descriptions :column="2" border size="small" style="margin-top:8px">
          <el-descriptions-item :label="$t('artifacts.recommendedTestMode')">{{ testModeText(selected) }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.recommendedEndpoint')">{{ recommendedEndpoint(selected) }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions v-if="selected?.default_test_mode" :column="1" border size="small" style="margin-top:8px">
          <el-descriptions-item :label="$t('artifacts.configuredTestMode')">
            <el-tag size="small" type="primary">{{ testModeText({ default_test_mode: selected.default_test_mode }) }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- GGUF metadata -->
        <template v-if="isGGUF && detailMeta">
          <h4 style="margin-top:12px">{{ $t('artifacts.ggufMetadata') }}</h4>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item v-if="detailMeta.embedding_length" :label="$t('artifacts.embeddingLength')">{{ detailMeta.embedding_length }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.block_count" :label="$t('artifacts.blockCount')">{{ detailMeta.block_count }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.vocab_size" :label="$t('artifacts.vocabSize')">{{ detailMeta.vocab_size }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.head_count" :label="$t('artifacts.headCount')">{{ detailMeta.head_count }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.head_count_kv" :label="$t('artifacts.headCountKV')">{{ detailMeta.head_count_kv }}</el-descriptions-item>
          </el-descriptions>
        </template>

        <!-- HF metadata -->
        <template v-if="isHF && detailMeta">
          <h4 style="margin-top:12px">{{ $t('artifacts.hfMetadata') }}</h4>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item v-if="detailMeta.model_type" :label="$t('artifacts.modelType')">{{ detailMeta.model_type }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.architectures" :label="$t('artifacts.architecture')">{{ Array.isArray(detailMeta.architectures) ? detailMeta.architectures.join(', ') : detailMeta.architectures }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.torch_dtype" :label="$t('artifacts.torchDtype')">{{ detailMeta.torch_dtype }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.max_position_embeddings" :label="$t('artifacts.maxPositionEmbeddings')">{{ detailMeta.max_position_embeddings }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.rope_scaling" :label="$t('artifacts.ropeScaling')">{{ JSON.stringify(detailMeta.rope_scaling) }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.hidden_size" :label="$t('artifacts.hiddenSize')">{{ detailMeta.hidden_size }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.num_hidden_layers" :label="$t('artifacts.numHiddenLayers')">{{ detailMeta.num_hidden_layers }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.num_attention_heads" :label="$t('artifacts.numAttentionHeads')">{{ detailMeta.num_attention_heads }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.vocab_size" :label="$t('artifacts.vocabSize')">{{ detailMeta.vocab_size }}</el-descriptions-item>
            <el-descriptions-item v-if="detailMeta.quantization_config" :label="$t('artifacts.quantizationConfig')">{{ JSON.stringify(detailMeta.quantization_config) }}</el-descriptions-item>
          </el-descriptions>
        </template>

        <!-- Warnings -->
        <el-alert v-if="detailMeta?.warnings?.length" type="warning" :closable="false" style="margin-top:8px" :title="$t('artifacts.warnings')">
          <ul style="margin:0;padding-left:16px"><li v-for="w in detailMeta.warnings" :key="w">{{ w }}</li></ul>
        </el-alert>

        <h4 style="margin-top:16px">{{ $t('modelLocations.title') }}</h4>
        <el-button size="small" type="primary" @click="showAddLocation" style="margin-bottom:8px">{{ $t('modelLocations.addLocation') }}</el-button>
        <el-table :data="locations" stripe size="small">
          <el-table-column :label="$t('modelLocations.node')" width="240" show-overflow-tooltip><template #default="{ row }">{{ nodeLabel(row.node_id) }}</template></el-table-column>
          <el-table-column prop="absolute_path" :label="$t('modelLocations.path')" min-width="200" show-overflow-tooltip />
          <el-table-column prop="verification_status" :label="$t('modelLocations.status')" width="100" />
          <el-table-column prop="match_status" :label="$t('modelLocations.matchStatus')" width="110" />
          <el-table-column :label="$t('common.actions')" width="180">
            <template #default="{ row: loc }">
              <el-button size="small" @click="doRescan(loc)">{{ $t('modelLocations.rescan') }}</el-button>
              <el-button size="small" type="danger" @click="doDeleteLocation(loc)">{{ $t('common.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- Scanner Recognition Info (Phase C) -->
        <template v-if="scanMeta">
          <h4 style="margin-top:16px">{{ $t('artifacts.scanRecognition') }}</h4>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item v-if="scanMeta.kind" :label="$t('artifacts.kind')">
              <el-tag size="small" :type="scanMeta.kind === 'directory' ? 'success' : scanMeta.kind === 'file' ? 'warning' : 'info'">{{ kindText(scanMeta.kind) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="scanMeta.deployable !== undefined" :label="$t('artifacts.deployable')">
              <el-tag size="small" :type="scanMeta.deployable ? 'success' : 'danger'">{{ scanMeta.deployable ? $t('common.yes') : $t('common.no') }}</el-tag>
              <span v-if="!scanMeta.deployable && scanMeta.unsupported_reason" style="margin-left:8px;color:#f56c6c">{{ scanMeta.unsupported_reason }}</span>
            </el-descriptions-item>
            <el-descriptions-item v-if="scanMeta.requires_base_model" :label="$t('artifacts.requiresBaseModel')">
              <el-tag size="small" type="warning">{{ $t('common.yes') }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="scanMeta.confidence" :label="$t('artifacts.confidence')">
              <el-tag size="small">{{ scanMeta.confidence }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="scanMeta.recommended_backends?.length" :label="$t('artifacts.recommendedBackends')" :span="2">
              <el-tag v-for="b in scanMeta.recommended_backends" :key="b" size="small" type="success" style="margin-right:4px">{{ b }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="scanMeta.evidence?.length" :label="$t('artifacts.evidence')" :span="2">
              <el-tag v-for="e in scanMeta.evidence" :key="e" size="small" type="info" style="margin-right:4px;margin-bottom:4px">{{ e }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="scanMeta.scan_root" :label="$t('artifacts.scanRoot')" :span="2">{{ scanMeta.scan_root }}</el-descriptions-item>
          </el-descriptions>
        </template>
      </template>
    </el-drawer>

    <!-- Wizard Dialog -->
    <el-dialog v-model="wizardVisible" :title="$t('modelWizard.title')" width="800px" :close-on-click-modal="false">
      <el-steps :active="wizardStep" finish-status="success" simple style="margin-bottom:20px">
        <el-step :title="$t('modelWizard.selectNode')" />
        <el-step :title="$t('modelWizard.browseDir')" />
        <el-step :title="$t('modelWizard.scanModel')" />
      </el-steps>
      <!-- Step 1: Select node -->
      <div v-if="wizardStep === 0">
        <NodeSelectorTable
          :nodes="wizardNodes"
          :loading="wizardNodesLoading"
          :error="wizardNodesError"
          :label="$t('nodeSelector.selectModelNode')"
          :hide-refresh="false"
          @select="onWizNodeSelect"
          @refresh="loadWizardNodes"
        />
        <div style="margin-top:12px;text-align:right"><el-button type="primary" :disabled="!wizardNodeId" @click="wizardStep=1">{{ $t('common.next') }}</el-button></div>
      </div>
      <!-- Step 2: File browser -->
      <div v-if="wizardStep === 1">
        <RemoteFileBrowser :node-id="wizardNodeId" @select="onFileSelect" />
        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=0">{{ $t('common.prev') }}</el-button>
          <el-button type="primary" :disabled="!wizardSelectedEntry" @click="doScan">{{ $t('modelWizard.scanModel') }}</el-button>
        </div>
      </div>
      <!-- Step 3: Scan results & save -->
      <div v-if="wizardStep === 2" v-loading="wizardScanning">
        <el-alert v-if="scanResult?.error && !scanResult?.candidates?.length" type="error" :title="scanResult.error" show-icon style="margin-bottom:12px" />
        <el-alert v-if="scanResult && !scanResult.error && !scanResult.candidates?.length" type="warning" :title="$t('modelWizard.noModelFound')" show-icon style="margin-bottom:12px" />

        <!-- Scan status info -->
        <el-alert v-if="scanResult && !scanResult.error && scanResult.candidates?.length > 0" type="success" :closable="false" style="margin-bottom:12px">
          <template #title>
            {{ $t('modelWizard.scanComplete') }}
            <template v-if="scanResult.candidates.length === 1 && scanResult.candidates[0].auto_selected">
              — {{ $t('modelWizard.autoSelected') }}:
              <el-tag size="small" :type="scanResult.candidates[0].format === 'huggingface' ? 'success' : 'warning'" style="margin-left:4px">
                {{ scanResult.candidates[0].format === 'huggingface' ? 'HuggingFace' : scanResult.candidates[0].format?.toUpperCase() }}
              </el-tag>
              {{ scanResult.candidates[0].path?.split('/').pop() }}
            </template>
            <template v-else>
              — {{ $t('modelWizard.scanSummary', { n: scanResult.candidates.length }) }}
            </template>
          </template>
        </el-alert>

        <!-- Scan directory info -->
        <div v-if="scanResult?.scan_root" style="margin-bottom:8px;font-size:13px;color:#909399">
          {{ $t('modelWizard.scanDirectory') }}: {{ scanResult.scan_root }}
        </div>

        <!-- Multi-candidate selection -->
        <div v-if="scanResult?.candidates?.length > 1" style="margin-bottom:12px">
          <div style="font-weight:500;margin-bottom:8px">
            {{ $t('modelWizard.selectCandidate') }}
            <el-tag v-if="hasHF" size="small" type="success" style="margin-left:8px">{{ $t('modelWizard.directoryModel') }}</el-tag>
            <el-tag v-if="hasGGUF" size="small" type="warning" style="margin-left:4px">{{ $t('modelWizard.fileModel') }}</el-tag>
          </div>
          <el-radio-group v-model="selectedCandidateIdx" @change="onCandidateSelect">
            <el-radio v-for="(c, idx) in scanResult.candidates" :key="idx" :value="idx" style="display:block;margin:4px 0">
              <el-tag size="small" :type="c.format === 'huggingface' ? 'success' : 'warning'" style="margin-right:8px">
                {{ c.format === 'huggingface' ? $t('modelWizard.formatHF') : c.format?.toUpperCase() }}
              </el-tag>
              {{ c.path?.split('/').pop() }}
              <template v-if="c.detected_metadata?.quantization">
                <el-tag size="small" type="warning" style="margin-left:4px">{{ c.detected_metadata.quantization }}</el-tag>
              </template>
              <template v-if="c.detected_metadata?.context_length">
                <span style="margin-left:8px;font-size:12px;color:#909399">{{ $t('modelWizard.contextLength') }}: {{ c.detected_metadata.context_length }}</span>
              </template>
              <template v-if="c.format === 'huggingface'">
                <span style="margin-left:8px;font-size:12px;color:#67c23a">{{ $t('modelWizard.directoryModelHint') }}</span>
              </template>
            </el-radio>
          </el-radio-group>
        </div>

        <!-- Single candidate detail -->
        <el-descriptions v-if="activeCandidate && !scanResult?.error" :column="2" border size="small">
          <el-descriptions-item :label="$t('modelWizard.modelName')">
            <span>{{ wizardModelName }}</span>
            <el-tag size="small" type="info" style="margin-left:8px">{{ $t('modelWizard.nameHint') }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.displayName')">
            <el-input v-model="wizardDisplayName" size="small" />
          </el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.modelFormat')">{{ activeCandidate.format || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.architecture')">{{ activeCandidate.detected_metadata?.architecture || activeCandidate.detected_metadata?.architectures || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.size')">{{ activeCandidate.size_label || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.path')">{{ activeCandidate.path || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('modelWizard.type')">{{ activeCandidate.path_type || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="activeCandidate.detected_metadata?.context_length" :label="$t('modelWizard.contextLength')">{{ activeCandidate.detected_metadata.context_length }}</el-descriptions-item>
          <el-descriptions-item v-if="activeCandidate.detected_metadata?.quantization" :label="$t('artifacts.quantization')">{{ activeCandidate.detected_metadata.quantization }}</el-descriptions-item>
          <el-descriptions-item :label="$t('artifacts.capabilities')" :span="2">
            <el-tag v-for="cap in wizardCapabilities" :key="cap.id" size="small" class="cap-tag">{{ capabilityText(cap) }}</el-tag>
            <span v-if="wizardCapabilities.length === 0">-</span>
          </el-descriptions-item>
        </el-descriptions>

        <!-- Warnings -->
        <el-alert v-if="activeCandidate?.warnings?.length" type="warning" :closable="false" style="margin-top:8px">
          <ul style="margin:0;padding-left:16px"><li v-for="w in activeCandidate.warnings" :key="w">{{ w }}</li></ul>
        </el-alert>

        <div style="margin-top:12px;text-align:right">
          <el-button @click="wizardStep=1">{{ $t('common.prev') }}</el-button>
          <el-button @click="doScan" :loading="wizardScanning">{{ $t('modelWizard.rescan') }}</el-button>
          <el-button type="primary" :disabled="!activeCandidate || !!scanResult?.error" @click="doWizardSave" :loading="wizardSaving">{{ $t('modelWizard.createAndSave') }}</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- Add Location Dialog -->
    <el-dialog v-model="addLocVisible" :title="$t('modelLocations.addLocation')" width="600px">
      <el-select v-model="addLocNodeId" :placeholder="$t('modelWizard.selectNode')" style="width:100%;margin-bottom:8px" filterable>
        <el-option v-for="n in nodeLabelItems" :key="n.id" :label="n.label" :value="n.id" />
      </el-select>
      <RemoteFileBrowser v-if="addLocNodeId" :node-id="addLocNodeId" @select="(e:any) => { addLocPath = e.relative_path; addLocSelected = e }" />
      <template #footer>
        <el-button @click="addLocVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!addLocPath" @click="doAddLocation" :loading="addLocSaving">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { apiClient } from '@/api/client'
import { useNodeLabels } from '@/composables/useNodeLabels'
import RemoteFileBrowser from '@/components/RemoteFileBrowser.vue'
import { useWizardAutoAdvance } from '@/composables/useWizardAutoAdvance'
import { capabilityLabel, inferModelCapabilities, recommendedTestMode, testModeLabel } from '@/utils/modelCapabilities.js'
import NodeSelectorTable from '@/components/common/NodeSelectorTable.vue'
const { loadNodes: loadNodeLabels, nodes: nodeLabelItems, nodeLabel } = useNodeLabels()
const { t, locale } = useI18n()

const loading = ref(false); const saving = ref(false)
const items = ref<any[]>([]); const dialogVisible = ref(false); const detailVisible = ref(false); const selected = ref<any>(null); const locations = ref<any[]>([])
const form = ref<Record<string, any>>({ name: '', path: '', format: 'custom', task_type: 'chat', architecture: 'custom', size_label: '', quantization: 'unknown', source_type: 'local_path', display_name: '' })
let editingId = ''

// Wizard state
const wizardVisible = ref(false); const wizardStep = ref(0)
const wizardNodeId = ref(''); const wizardSelectedEntry = ref<any>(null)
const wizardNodes = ref<any[]>([]); const wizardNodesLoading = ref(false); const wizardNodesError = ref('')
const wizardScanning = ref(false); const wizardSaving = ref(false)
const scanResult = ref<any>(null); const wizardModelName = ref(''); const wizardDisplayName = ref('')
const selectedCandidateIdx = ref(0)
const activeCandidate = ref<any>(null)

const { onSelectAutoNext: onWizAutoNext } = useWizardAutoAdvance(wizardStep, () => { wizardStep.value++ })

async function loadWizardNodes() {
  wizardNodesLoading.value = true
  wizardNodesError.value = ''
  try {
    wizardNodes.value = await apiClient.get('/nodes')
  } catch (e: any) {
    wizardNodesError.value = e?.message || 'Failed to load nodes'
  } finally {
    wizardNodesLoading.value = false
  }
}

function onWizNodeSelect(node: any) {
  if (node) wizardNodeId.value = node.id
}

// Add location state
const addLocVisible = ref(false); const addLocNodeId = ref(''); const addLocPath = ref(''); const addLocSelected = ref<any>(null); const addLocSaving = ref(false)

const formatOptions = ['gguf', 'safetensors', 'huggingface', 'pt', 'onnx', 'other']
const quantOptions = ['Q4_K_M', 'Q5_K_M', 'Q8_0', 'FP16', 'BF16', 'FP8', 'INT8', 'INT4', 'none', 'other']

// Capability edit state
const editCapabilities = ref<string[]>([])
const editDefaultTestMode = ref('auto')
const editTaskType = ref('chat')
const parameterDefaults = ref<any[]>([])
const parameterDefaultsText = ref('')
const TASK_TYPE_OPTIONS = [
  { value: 'chat', labelKey: 'artifacts.task_chat' },
  { value: 'completion', labelKey: 'artifacts.task_completion' },
  { value: 'embedding', labelKey: 'artifacts.task_embedding' },
  { value: 'rerank', labelKey: 'artifacts.task_rerank' },
  { value: 'vision_chat', labelKey: 'artifacts.task_visionChat' },
  { value: 'adapter', labelKey: 'artifacts.task_adapter' },
  { value: 'unknown', labelKey: 'artifacts.task_unknown' },
]
const CAPABILITY_OPTIONS = [
  { value: 'chat', labelKey: 'artifacts.capability_chat' },
  { value: 'completion', labelKey: 'artifacts.capability_completion' },
  { value: 'embedding', labelKey: 'artifacts.capability_embedding' },
  { value: 'rerank', labelKey: 'artifacts.capability_rerank' },
  { value: 'vision', labelKey: 'artifacts.capability_vision' },
  { value: 'tool_calling', labelKey: 'artifacts.capability_tool_calling' },
  { value: 'structured_output', labelKey: 'artifacts.capability_structured_output' },
]
const TEST_MODE_OPTIONS = [
  { value: 'auto', labelKey: 'artifacts.testMode_auto' },
  { value: 'chat', labelKey: 'artifacts.testMode_chat' },
  { value: 'completion', labelKey: 'artifacts.testMode_completion' },
  { value: 'embedding', labelKey: 'artifacts.testMode_embedding' },
  { value: 'rerank', labelKey: 'artifacts.testMode_rerank' },
]

const hasPersistedCapabilities = computed(() => {
  return Array.isArray(selected.value?.capabilities) && selected.value.capabilities.length > 0
})

// Scanner metadata from first model location (Phase C).
const scanMeta = computed(() => {
  const loc = locations.value?.[0]
  if (!loc?.discovered_metadata_json) return null
  return loc.discovered_metadata_json
})

function kindText(kind: string): string {
  const keyMap: Record<string, string> = { directory: 'artifacts.kind_directory', file: 'artifacts.kind_file', adapter: 'artifacts.kind_adapter' }
  return t(keyMap[kind] || kind)
}

function taskTypeText(task: string): string {
  const keyMap: Record<string, string> = { chat: 'artifacts.task_chat', completion: 'artifacts.task_completion', embedding: 'artifacts.task_embedding', rerank: 'artifacts.task_rerank', vision_chat: 'artifacts.task_visionChat', adapter: 'artifacts.task_adapter', unknown: 'artifacts.task_unknown' }
  return t(keyMap[task] || task)
}

// Scan result candidate type checks.
const hasHF = computed(() => scanResult.value?.candidates?.some((c: any) => c.format === 'huggingface'))
const hasGGUF = computed(() => scanResult.value?.candidates?.some((c: any) => c.format === 'gguf'))

// Computed metadata from selected artifact + first location
const detailMeta = computed(() => {
  const loc = locations.value?.[0]
  if (!loc) return null
  // Prefer discovered_metadata_json from location, fall back to artifact fields
  const meta = loc.discovered_metadata_json || {}
  if (!meta.architecture && selected.value?.architecture && selected.value.architecture !== 'custom') {
    meta.architecture = selected.value.architecture
  }
  if (!meta.quantization && selected.value?.quantization && selected.value.quantization !== 'unknown') {
    meta.quantization = selected.value.quantization
  }
  if (!meta.context_length && selected.value?.default_context_length) {
    meta.context_length = selected.value.default_context_length
  }
  return meta
})
const isGGUF = computed(() => selected.value?.format === 'gguf')
const isHF = computed(() => selected.value?.format === 'huggingface' || selected.value?.format === 'safetensors')
const detailPathType = computed(() => locations.value?.[0]?.path_type || '')
const detailFileSize = computed(() => {
  const bytes = detailMeta.value?.file_size_bytes || locations.value?.[0]?.size_bytes
  if (!bytes || bytes === 0) return ''
  return formatBytesHuman(bytes)
})
const detailParamCount = computed(() => detailMeta.value?.parameter_count || '')
const detailCtxLen = computed(() => detailMeta.value?.context_length || detailMeta.value?.max_position_embeddings || selected.value?.default_context_length || '')
const detailCapabilities = computed(() => capabilitiesFor(selected.value ? { ...selected.value, locations: locations.value } : null))
const wizardCapabilities = computed(() => {
  if (!activeCandidate.value) return []
  return capabilitiesFor({
    name: wizardModelName.value,
    display_name: wizardDisplayName.value,
    format: activeCandidate.value.format,
    task_type: 'chat',
    architecture: activeCandidate.value.detected_metadata?.architecture,
    discovered_metadata_json: activeCandidate.value.detected_metadata || {},
  })
})

function formatBytesHuman(bytes: number): string {
  if (!bytes || bytes === 0) return ''
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let i = 0; let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return size.toFixed(1) + ' ' + units[i]
}

function capabilitiesFor(model: any): any[] {
  if (!model) return []
  return inferModelCapabilities(model)
}

function capabilityText(cap: any): string {
  return capabilityLabel(cap, locale.value)
}

function capabilitySourceText(source: string): string {
  // Map new capability source values to i18n keys.
  const sourceKeyMap: Record<string, string> = {
    explicit: 'artifacts.capabilitySource_explicit',
    metadata: 'artifacts.capabilitySource_metadata',
    inferred: 'artifacts.capabilitySource_inferred',
    scan: 'artifacts.capabilitySource_scan',
    user_override: 'artifacts.capabilitySource_userOverride',
    backend_probe: 'artifacts.capabilitySource_backendProbe',
  }
  const key = sourceKeyMap[source] || `artifacts.capabilitySource_${source || 'unknown'}`
  const val = t(key)
  return val === key ? (source || '-') : val
}

function capabilityConfidenceText(confidence: string): string {
  const key = `artifacts.capabilityConfidence_${confidence || 'unknown'}`
  const val = t(key)
  return val === key ? (confidence || '-') : val
}

function testModeText(model: any): string {
  return testModeLabel(recommendedTestMode(model), locale.value)
}

function recommendedEndpoint(model: any): string {
  const mode = recommendedTestMode(model)
  if (mode === 'chat') return '/v1/chat/completions'
  if (mode === 'completion') return '/v1/completions'
  return t('artifacts.endpointAuto')
}

onMounted(async () => { await refresh(); loadNodeLabels() })

async function refresh() {
  loading.value = true
  try { items.value = await apiClient.get('/api/v1/model-artifacts') } catch (e: any) { console.error(e) }
  loading.value = false
}
async function loadNodesLocal() { loadNodeLabels() }

function showCreate() { editingId = ''; form.value = { name: '', path: '', format: 'custom', task_type: 'chat', architecture: 'custom', size_label: '', quantization: 'unknown', source_type: 'local_path', display_name: '' }; editCapabilities.value = []; editDefaultTestMode.value = 'auto'; editTaskType.value = 'chat'; parameterDefaultsText.value = ''; dialogVisible.value = true }
function showEdit(row: any) {
  editingId = row.id; Object.assign(form.value, row)
  editCapabilities.value = Array.isArray(row.capabilities) ? [...row.capabilities] : []
  editDefaultTestMode.value = row.default_test_mode || 'auto'
  editTaskType.value = row.task_type || 'chat'
  // Convert parameter_defaults array to text lines
  const pd = Array.isArray(row.parameter_defaults) ? row.parameter_defaults : []
  parameterDefaultsText.value = pd.map((p: any) => {
    const cliName = p.cli_name || p.key || ''
    const val = p.value != null ? String(p.value) : ''
    return val ? `${cliName} ${val}` : cliName
  }).filter(Boolean).join('\n')
  dialogVisible.value = true
}

async function doSave() {
  saving.value = true
  try {
    if (!form.value.display_name) form.value.display_name = form.value.name
    const payload: any = { ...form.value }
    payload.capabilities = editCapabilities.value
    payload.default_test_mode = editDefaultTestMode.value
    payload.task_type = editTaskType.value
    // Convert text lines to structured parameter_defaults array
    const lines = parameterDefaultsText.value.split('\n').map((l: string) => l.trim()).filter(Boolean)
    payload.parameter_defaults = lines.map((line: string) => {
      const parts = line.split(/\s+/)
      const cliName = parts[0] || ''
      const value = parts.slice(1).join(' ')
      return { key: cliName.replace(/^-+/, ''), cli_name: cliName, value, type: 'string', enabled: true }
    })
    if (editingId) await apiClient.patch(`/api/v1/model-artifacts/${editingId}`, payload)
    else await apiClient.post('/api/v1/model-artifacts', payload)
    ElMessage.success(t('artifacts.saved')); dialogVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  saving.value = false
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(t('artifacts.deleteConfirm', { name: row.name }), t('common.confirm'), { type: 'warning' })
    await apiClient.delete(`/api/v1/model-artifacts/${row.id}`)
    ElMessage.success(t('artifacts.deleted')); await refresh()
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}

async function showDetail(row: any) {
  selected.value = row
  try { locations.value = await apiClient.get(`/api/v1/model-artifacts/${row.id}`).then((r: any) => r.locations || []) } catch { locations.value = [] }
  detailVisible.value = true
}

// ---- Wizard ----
function startWizard() { wizardVisible.value = true; wizardStep.value = 0; wizardNodeId.value = ''; wizardSelectedEntry.value = null; scanResult.value = null; wizardModelName.value = ''; wizardDisplayName.value = ''; selectedCandidateIdx.value = 0; activeCandidate.value = null; loadWizardNodes() }
function onFileSelect(entry: any) {
  wizardSelectedEntry.value = entry
  wizardModelName.value = entry.name
  wizardDisplayName.value = entry.name
}

function onCandidateSelect(idx: number) {
  selectedCandidateIdx.value = idx
  const c = scanResult.value?.candidates?.[idx]
  if (c) {
    activeCandidate.value = c
    const name = c.path?.split('/').pop() || ''
    wizardModelName.value = name
    wizardDisplayName.value = name
  }
}

async function doScan() {
  if (!wizardSelectedEntry.value || !wizardNodeId.value) return
  wizardScanning.value = true; wizardStep.value = 2
  try {
    const entry = wizardSelectedEntry.value
    const root = entry.root || ''
    const relPath = entry.relative_path || entry.name
    const resp = await apiClient.post(`/nodes/${wizardNodeId.value}/model-paths/scan`, { root_id: entry.root_id, root, relative_path: relPath, path_type: entry.path_type || (entry.is_dir ? 'directory' : 'file') })
    scanResult.value = resp
    // Handle new candidate-based response
    if (resp.candidates?.length) {
      // Auto-select: pick auto_selected first, then prefer HF directory over GGUF files,
      // then fall back to the first candidate (WEB-AI-RC-006).
      let autoIdx = 0
      for (let i = 0; i < resp.candidates.length; i++) {
        if (resp.candidates[i].auto_selected) { autoIdx = i; break }
      }
      // If no auto_selected, prefer HuggingFace directory model over GGUF files.
      if (!resp.candidates[autoIdx]?.auto_selected) {
        const hfIdx = resp.candidates.findIndex((c: any) => c.format === 'huggingface')
        if (hfIdx >= 0) { autoIdx = hfIdx }
      }
      selectedCandidateIdx.value = autoIdx
      activeCandidate.value = resp.candidates[autoIdx]
      const name = resp.candidates[autoIdx].path?.split('/').pop() || ''
      wizardModelName.value = name
      wizardDisplayName.value = name
    } else if (resp.discovered_name) {
      // Legacy flat response fallback
      activeCandidate.value = resp
      wizardModelName.value = resp.discovered_name
      wizardDisplayName.value = resp.discovered_name
    }
  } catch (e: any) { scanResult.value = { error: e?.message || t('modelWizard.scanFailed') } }
  wizardScanning.value = false
}

async function doWizardSave() {
  if (!activeCandidate.value) return
  wizardSaving.value = true
  try {
    const c = activeCandidate.value
    // Pass candidate's task, capabilities, default_test_mode from scanner (Phase A+B1).
    const caps = c.capabilities || []
    const dtm = c.default_test_mode || 'auto'
    const task = c.task || 'chat'

    const artifact = await apiClient.post('/api/v1/model-artifacts', {
      name: wizardModelName.value,
      path: c.path || scanResult.value?.absolute_path,
      format: c.format || 'huggingface',
      task_type: task,
      size_label: c.size_label || scanResult.value?.size_label || '',
      source_type: 'local_path',
      display_name: wizardDisplayName.value || wizardModelName.value,
      architecture: c.detected_metadata?.architecture || 'custom',
      default_context_length: c.detected_metadata?.context_length || 0,
      quantization: c.detected_metadata?.quantization || 'unknown',
      capabilities: caps,
      default_test_mode: dtm,
    })
    // Derive the model location path from the candidate (the specific discovered file).
    // The scan root and the candidate path are needed to compute model_root and relative_path
    // that point to the exact .gguf file, not just the scan directory (WEB-AI-RC-006).
    const candidatePath: string = c.path || scanResult.value?.absolute_path || ''
    const scanRoot = scanResult.value?.model_root || scanResult.value?.root || wizardSelectedEntry.value?.root || ''
    const candidatePathType = c.path_type || (candidatePath.endsWith('.gguf') ? 'file' : 'directory')

    // Compute model_root and relative_path from the candidate's specific file path.
    let locModelRoot = scanRoot
    let locRelativePath = ''
    if (candidatePath && scanRoot && candidatePath.startsWith(scanRoot)) {
      locRelativePath = candidatePath.slice(scanRoot.length).replace(/^\//, '')
    }
    // Fallback: use scan-level relative_path if the candidate path doesn't start with root.
    if (!locRelativePath) {
      locModelRoot = scanResult.value?.model_root || scanResult.value?.root || wizardSelectedEntry.value?.root
      locRelativePath = scanResult.value?.relative_path || wizardSelectedEntry.value?.relative_path
    }

    // Enrich discovered_metadata_json with scanner metadata (Phase A+B1).
    const scanMeta: any = { ...(c.detected_metadata || {}) }
    if (c.kind) scanMeta.kind = c.kind
    if (c.task) scanMeta.task = c.task
    if (c.deployable !== undefined) scanMeta.deployable = c.deployable
    if (c.requires_base_model !== undefined) scanMeta.requires_base_model = c.requires_base_model
    if (c.recommended_backends?.length) scanMeta.recommended_backends = c.recommended_backends
    if (c.confidence) scanMeta.confidence = c.confidence
    if (c.evidence?.length) scanMeta.evidence = c.evidence
    if (c.unsupported_reason) scanMeta.unsupported_reason = c.unsupported_reason

    await apiClient.post(`/model-artifacts/${artifact.id}/locations`, {
      node_id: wizardNodeId.value,
      root_id: scanResult.value?.root_id || wizardSelectedEntry.value?.root_id,
      model_root: locModelRoot,
      relative_path: locRelativePath || wizardSelectedEntry.value?.relative_path,
      absolute_path: candidatePath,
      path_type: candidatePathType,
      verification_status: 'verified',
      match_status: 'exact_match',
      discovered_metadata_json: scanMeta,
    })
    ElMessage.success(t('artifacts.created')); wizardVisible.value = false; await refresh()
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  wizardSaving.value = false
}

// ---- Location management ----
function showAddLocation() { addLocVisible.value = true; addLocNodeId.value = ''; addLocPath.value = '' }
async function doAddLocation() {
  if (!selected.value || !addLocNodeId.value || !addLocPath.value) return
  addLocSaving.value = true
  try {
    await apiClient.post(`/model-artifacts/${selected.value.id}/locations`, {
      node_id: addLocNodeId.value, root_id: addLocSelected.value.root_id,
      model_root: addLocSelected.value.root,
      relative_path: addLocSelected.value.relative_path,
      path_type: addLocSelected.value.path_type || 'directory',
      verification_status: 'verified', match_status: 'exact_match',
    })
    ElMessage.success(t('modelLocations.added')); addLocVisible.value = false
    await showDetail(selected.value)
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
  addLocSaving.value = false
}
async function doRescan(loc: any) {
  try {
    await apiClient.post(`/model-artifacts/${selected.value.id}/locations/${loc.id}/rescan`)
    ElMessage.success(t('modelLocations.rescanned')); await showDetail(selected.value)
  } catch (e: any) { ElMessage.error(e?.message || t('common.failed')) }
}
async function doDeleteLocation(loc: any) {
  try {
    await ElMessageBox.confirm(t('modelLocations.deleteConfirm'), t('common.confirm'), { type: 'warning' })
    await apiClient.delete(`/model-artifacts/${selected.value.id}/locations/${loc.id}`)
    ElMessage.success(t('modelLocations.deleted')); await showDetail(selected.value)
  } catch (e: any) { if (e !== 'cancel') ElMessage.error(e?.message || t('common.failed')) }
}
</script>

<style scoped>
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.page-header h2 { margin: 0; }
.cap-tag { margin-right: 4px; margin-bottom: 4px; }
</style>
