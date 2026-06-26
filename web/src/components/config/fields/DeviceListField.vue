<template>
  <el-input v-model="model" type="textarea" :rows="2" :disabled="disabled" placeholder="/dev/nvidia0" @input="$emit('change')" />
</template>

<script setup lang="ts">
const model = defineModel<string>()
defineProps<{ disabled?: boolean }>()
defineEmits<{ change: [] }>()
</script>
