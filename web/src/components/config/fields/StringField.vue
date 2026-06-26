<template>
  <el-input v-model="model" size="small" :disabled="disabled" @input="$emit('change')" />
</template>

<script setup lang="ts">
const model = defineModel<any>()
defineProps<{ disabled?: boolean }>()
defineEmits<{ change: [] }>()
</script>
