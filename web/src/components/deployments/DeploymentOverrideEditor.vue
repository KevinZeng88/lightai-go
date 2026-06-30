<template>
  <div>
    <el-alert type="info" :closable="false" style="margin-bottom:12px">
      {{ $t('deployments.overrideHint') }}
    </el-alert>
    <ConfigEditView
      v-if="editView"
      :model-value="editView"
      @update:patch="onUpdate"
    />
    <el-empty v-else :description="$t('common.noData')" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getConfigEditView } from '@/api/configEdit'
import ConfigEditView from '@/components/config/ConfigEditView.vue'
import type { ConfigEditPatch, ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'

const props = defineProps<{
  nbrConfigSet: Record<string, any> | null
  nbrId?: string
}>()

const emit = defineEmits<{
  'update:overrides': [value: Record<string, any>]
  'update:patch': [value: ConfigEditPatch | null]
}>()

const editView = ref<ConfigEditViewModel | null>(null)

watch(() => props.nbrId, loadView, { immediate: true })

async function loadView() {
  editView.value = null
  emit('update:patch', null)
  if (!props.nbrId) return
  editView.value = await getConfigEditView({
    object_kind: 'node_backend_runtime',
    object_id: props.nbrId,
    layer: 'deployment',
    mode: 'deployment_override',
    view_level: 'normal',
  })
}

function onUpdate(patch: ConfigEditPatch) {
  emit('update:patch', patch)
  emit('update:overrides', { editable_config_patch: patch })
}
</script>
