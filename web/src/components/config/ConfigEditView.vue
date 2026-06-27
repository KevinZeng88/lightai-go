<template>
  <div
    v-if="localView"
    class="config-edit-view"
    data-testid="config-edit-view"
    :data-object-kind="localView.object_kind"
    :data-layer="localView.layer"
    :data-object-id="localView.object_id"
  >
    <ConfigSection
      v-for="section in sortedSections(localView)"
      :key="section.key"
      :section="section"
      :readonly="readonly || localView.readonly"
      @change="emitPatch"
    />
  </div>
  <el-empty v-else :description="$t('common.noData')" />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ConfigSection from './ConfigSection.vue'
import { buildConfigEditPatch, cloneEditView, sortedSections, type ConfigEditPatch, type ConfigEditView } from '@/utils/configEditView'

const props = withDefaults(defineProps<{
  modelValue: ConfigEditView | null
  readonly?: boolean
}>(), {
  readonly: false,
})

const emit = defineEmits<{
  'update:patch': [value: ConfigEditPatch]
}>()

const localView = ref<ConfigEditView | null>(cloneEditView(props.modelValue))

watch(() => props.modelValue, (value) => {
  localView.value = cloneEditView(value)
}, { deep: true })

function emitPatch() {
  if (!localView.value) return
  emit('update:patch', buildConfigEditPatch(localView.value))
}
</script>

<style scoped>
.config-edit-view {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
