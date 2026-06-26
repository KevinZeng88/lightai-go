<template>
  <el-input v-model="model" type="textarea" :rows="3" :disabled="disabled" placeholder="KEY=value" @input="$emit('change')" />
</template>

<script setup lang="ts">
const model = defineModel<string>()
defineProps<{ disabled?: boolean }>()
defineEmits<{ change: [] }>()
</script>
