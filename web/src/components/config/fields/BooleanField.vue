<template>
  <el-switch v-model="model" :disabled="disabled" @change="$emit('change')" />
</template>

<script setup lang="ts">
const model = defineModel<boolean>()
defineProps<{ disabled?: boolean }>()
defineEmits<{ change: [] }>()
</script>
