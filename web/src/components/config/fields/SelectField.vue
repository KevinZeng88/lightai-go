<template>
  <el-select v-model="model" :multiple="multiple" size="small" :disabled="disabled" @change="$emit('change')">
    <el-option v-for="option in options" :key="String(option.value)" :label="option.label" :value="option.value" />
  </el-select>
</template>

<script setup lang="ts">
const model = defineModel<any>()
withDefaults(defineProps<{ disabled?: boolean, multiple?: boolean, options?: Array<{ label: string, value: any }> }>(), {
  options: () => [],
  multiple: false,
})
defineEmits<{ change: [] }>()
</script>
