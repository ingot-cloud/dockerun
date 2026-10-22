<template>
  <div class="volume-key-values">
    <div v-for="(pair, index) in modelValue" :key="index" class="volume-pair">
      <el-input v-model="pair.key" :aria-label="`${label} ${index + 1} 名称`" placeholder="名称" :disabled="disabled" />
      <el-input v-model="pair.value" :aria-label="`${label} ${index + 1} 值`" placeholder="值" :disabled="disabled" />
      <el-button class="volume-pair-remove" :aria-label="`移除${label} ${index + 1}`" :disabled="disabled" @click="emit('update:modelValue', modelValue.filter((_, i) => i !== index))"><PortainerIcon name="trash" /></el-button>
    </div>
    <el-button link :disabled="disabled" @click="emit('update:modelValue', [...modelValue, { key: '', value: '' }])"><PortainerIcon name="plus" />添加{{ label }}</el-button>
  </div>
</template>
<script setup lang="ts">
import PortainerIcon from '@/components/PortainerIcon.vue'
import type { VolumePair } from '@/utils/volumes'
defineProps<{ modelValue: VolumePair[]; label: string; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: VolumePair[]] }>()
</script>
