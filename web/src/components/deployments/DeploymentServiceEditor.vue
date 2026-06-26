<template>
  <div>
    <el-form label-position="top">
      <el-form-item :label="$t('deployments.hostPort')">
        <el-input-number v-model="hostPortModel" :min="1" :max="65535" style="width:100%" />
      </el-form-item>
      <el-form-item :label="$t('deployments.containerPort')">
        <el-input-number v-model="containerPortModel" :min="1" :max="65535" style="width:100%" />
      </el-form-item>
      <el-form-item :label="$t('deployments.servedModelName')">
        <el-input v-model="servedModelNameModel" placeholder="my-model-name" />
      </el-form-item>
      <el-form-item :label="$t('deployments.endpointPreview')">
        <el-input :model-value="endpointPreview" readonly />
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  hostPort: number
  containerPort: number
  servedModelName: string
}>()

const emit = defineEmits<{
  'update:hostPort': [value: number]
  'update:containerPort': [value: number]
  'update:servedModelName': [value: string]
}>()

const hostPortModel = computed({ get: () => props.hostPort, set: (v) => emit('update:hostPort', v) })
const containerPortModel = computed({ get: () => props.containerPort, set: (v) => emit('update:containerPort', v) })
const servedModelNameModel = computed({ get: () => props.servedModelName, set: (v) => emit('update:servedModelName', v) })

const endpointPreview = computed(() =>
  props.hostPort ? `http://<node-ip>:${props.hostPort}/v1` : ''
)
</script>
