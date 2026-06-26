<template>
  <el-input-number v-model="model" size="small" :disabled="disabled" @change="$emit('change')" />
</template>

<script setup lang="ts">
const model = defineModel<number | undefined>()
defineProps<{ disabled?: boolean }>()
defineEmits<{ change: [] }>()
</script>
