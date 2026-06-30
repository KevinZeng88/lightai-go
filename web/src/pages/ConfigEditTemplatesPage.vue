<template>
  <div class="template-page">
    <div class="page-header">
      <h2>{{ $t('configEdit.templates.title') }}</h2>
      <el-button :loading="loading" @click="loadTemplates">{{ $t('common.refresh') }}</el-button>
    </div>

    <el-row :gutter="12">
      <el-col :span="9">
        <el-table :data="templates" border size="small" height="640" @row-click="selectTemplate">
          <el-table-column prop="template_id" :label="$t('configEdit.templates.template')" min-width="220" />
          <el-table-column prop="applies_to.backend" :label="$t('configEdit.templates.backend')" width="100" />
          <el-table-column prop="source" :label="$t('configEdit.templates.source')" width="140" />
        </el-table>
      </el-col>
      <el-col :span="15">
        <el-empty v-if="!selected" :description="$t('configEdit.templates.selectTemplate')" />
        <div v-else class="detail">
          <div class="detail-actions">
            <div>
              <h3>{{ selected.template_id }}</h3>
              <div class="muted">
                {{ selected.metadata?.display_name || selected.template_id }}
              </div>
            </div>
            <div class="actions">
              <el-button v-if="selected.source === 'built_in'" @click="cloneSelected">{{ $t('configEdit.templates.clone') }}</el-button>
              <el-button type="primary" @click="validateSelected">{{ $t('configEdit.templates.validate') }}</el-button>
            </div>
          </div>

          <el-descriptions :column="2" border size="small">
            <el-descriptions-item :label="$t('configEdit.templates.backend')">{{ selected.applies_to?.backend || '-' }}</el-descriptions-item>
            <el-descriptions-item :label="$t('configEdit.templates.vendors')">{{ (selected.applies_to?.vendors || []).join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item :label="$t('configEdit.templates.runtime')">{{ selected.applies_to?.runtime_kind || '-' }}</el-descriptions-item>
            <el-descriptions-item :label="$t('configEdit.templates.source')">{{ selected.source || '-' }}</el-descriptions-item>
          </el-descriptions>

          <el-tabs v-model="activeTab" class="tabs">
            <el-tab-pane :label="$t('configEdit.templates.fields')" name="fields">
              <el-table :data="selected.fields || []" border size="small" height="360">
                <el-table-column prop="key" :label="$t('configEdit.templates.key')" min-width="220" />
                <el-table-column prop="section" :label="$t('configEdit.templates.section')" width="150" />
                <el-table-column prop="tier" :label="$t('configEdit.templates.tier')" width="100" />
                <el-table-column prop="view" :label="$t('configEdit.templates.view')" width="100" />
                <el-table-column prop="source.value_source" :label="$t('configEdit.templates.source')" width="140" />
                <el-table-column prop="enabled" :label="$t('common.enabled')" width="90" />
              </el-table>
            </el-tab-pane>
            <el-tab-pane :label="$t('configEdit.templates.components')" name="components">
              <el-table :data="selected.components || []" border size="small" height="300">
                <el-table-column prop="key" :label="$t('configEdit.templates.key')" min-width="220" />
                <el-table-column prop="renderer" :label="$t('configEdit.templates.renderer')" width="160" />
                <el-table-column prop="view" :label="$t('configEdit.templates.view')" width="100" />
                <el-table-column :label="$t('configEdit.templates.effects')" width="90">
                  <template #default="{ row }">{{ row.effects?.length || 0 }}</template>
                </el-table-column>
              </el-table>
            </el-tab-pane>
            <el-tab-pane :label="$t('configEdit.templates.diagnostics')" name="diagnostics">
              <el-input v-model="rawText" type="textarea" :rows="18" />
            </el-tab-pane>
            <el-tab-pane :label="$t('configEdit.templates.validation')" name="validation">
              <el-alert v-if="validation && validation.valid" :title="$t('configEdit.templates.templateValid')" type="success" show-icon />
              <el-table v-else :data="validation?.issues || []" border size="small">
                <el-table-column prop="severity" :label="$t('configEdit.templates.severity')" width="100" />
                <el-table-column prop="path" :label="$t('configEdit.templates.path')" width="240" />
                <el-table-column prop="reason" :label="$t('configEdit.templates.reason')" />
              </el-table>
            </el-tab-pane>
          </el-tabs>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { cloneConfigEditTemplate, listConfigEditTemplates, validateConfigEditTemplate } from '@/api/configEdit'

const { t } = useI18n()
const loading = ref(false)
const store = ref<any>({ templates: [] })
const selected = ref<any | null>(null)
const rawText = ref('')
const validation = ref<any | null>(null)
const activeTab = ref('fields')
const templates = computed(() => store.value?.templates || [])

async function loadTemplates() {
  loading.value = true
  try {
    store.value = await listConfigEditTemplates()
    if (!selected.value && templates.value.length) selectTemplate(templates.value[0])
  } finally {
    loading.value = false
  }
}

function selectTemplate(row: any) {
  selected.value = row
  validation.value = null
  activeTab.value = (row.fields || []).length ? 'fields' : 'components'
  rawText.value = JSON.stringify(row, null, 2)
}

async function validateSelected() {
  if (!rawText.value) return
  try {
    validation.value = await validateConfigEditTemplate(JSON.parse(rawText.value))
  } catch (error: any) {
    validation.value = { valid: false, issues: [{ severity: 'error', path: 'raw', reason: error?.message || 'Invalid JSON' }] }
  }
}

async function cloneSelected() {
  if (!selected.value?.template_id) return
  const cloned = await cloneConfigEditTemplate(selected.value.template_id)
  ElMessage.success(t('configEdit.templates.cloned'))
  await loadTemplates()
  const row = templates.value.find((item: any) => item.template_id === cloned.template_id)
  if (row) selectTemplate(row)
}

watch(selected, value => {
  rawText.value = value ? JSON.stringify(value, null, 2) : ''
})

onMounted(loadTemplates)
</script>

<style scoped>
.template-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.page-header,
.detail-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
h2,
h3 {
  margin: 0;
}
.detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.actions {
  display: flex;
  gap: 8px;
}
.muted {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin-top: 4px;
}
.tabs {
  margin-top: 4px;
}
</style>
