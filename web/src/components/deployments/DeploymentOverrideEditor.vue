<template>
  <div>
    <el-alert type="info" :closable="false" style="margin-bottom:12px">
      {{ $t('deployments.overrideHint') || 'Edit deployment-level overrides. Inherited values from the node runtime config are shown for reference.' }}
    </el-alert>
    <RuntimeParameterEditor
      v-if="props.nbrConfigSet"
      :model-value="editorModel"
      :layer="'deployment'"
      :show-source="true"
      :show-advanced="true"
      @update:model-value="onUpdate"
    />
    <el-empty v-else :description="$t('common.noData') || 'Select a node runtime config first'" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import RuntimeParameterEditor from '@/components/common/RuntimeParameterEditor.vue'

const props = defineProps<{
  nbrConfigSet: Record<string, any> | null
}>()

const emit = defineEmits<{ 'update:overrides': [value: Record<string, any>] }>()

const editorModel = ref<Record<string, any>>({ config_set: props.nbrConfigSet || {} })

watch(() => props.nbrConfigSet, (cs) => {
  editorModel.value = { config_set: cs || {} }
})

function onUpdate(val: Record<string, any>) {
  editorModel.value = val
  if (val.config_overrides) {
    emit('update:overrides', val.config_overrides)
  }
}
</script>
