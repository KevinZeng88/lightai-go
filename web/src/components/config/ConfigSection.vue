<template>
  <el-collapse :model-value="activeNames" @update:model-value="activeNames = normalizeActive($event)">
    <el-collapse-item :name="section.key">
      <template #title>
        <span>{{ section.label }}</span>
        <el-tag v-if="section.key === 'advanced_raw'" size="small" effect="plain" class="section-tag">advanced_raw</el-tag>
      </template>
      <p v-if="section.description" class="section-description">{{ section.description }}</p>
      <div class="section-fields">
        <ConfigField
          v-for="field in sortedFields(section)"
          :key="field.key"
          :field="field"
          :readonly="readonly"
          @change="$emit('change')"
        />
      </div>
    </el-collapse-item>
  </el-collapse>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ConfigField from './ConfigField.vue'
import { sortedFields, type ConfigEditSection } from '@/utils/configEditView'

const props = defineProps<{
  section: ConfigEditSection
  readonly?: boolean
}>()

defineEmits<{ change: [] }>()

const activeNames = ref<string[]>(props.section.collapsed ? [] : [props.section.key])

watch(() => props.section.key, () => {
  activeNames.value = props.section.collapsed ? [] : [props.section.key]
})

function normalizeActive(value: string | string[]) {
  return Array.isArray(value) ? value : [value]
}
</script>

<style scoped>
.section-tag { margin-left: 8px; }
.section-description { margin: 0 0 10px; color: var(--el-text-color-secondary); font-size: 12px; }
.section-fields { display: grid; grid-template-columns: 1fr; gap: 10px; }
</style>
