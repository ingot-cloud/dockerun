<template>
  <ol class="application-setup-steps" aria-label="应用首次发布步骤">
    <li v-for="(label,index) in labels" :key="label" :class="{done:index<step,current:index===step}" :aria-current="index===step?'step':undefined"><span aria-hidden="true">{{index<step?'✓':index+1}}</span>{{label}}<span class="application-step-state">{{index<step?'（已完成）':index===step?'（当前步骤）':''}}</span></li>
  </ol>
</template>
<script setup lang="ts">
defineProps<{step:number}>()
const labels=['创建应用','配置环境','首次发布']
</script>
<style scoped>
.application-setup-steps { display:flex; align-items:center; gap:8px; padding:0; margin:0; list-style:none; min-width:0; font-size:12px; color:#999; }
.application-setup-steps li { display:flex; align-items:center; gap:6px; white-space:nowrap; }
.application-setup-steps li + li::before { content:'›'; margin-right:2px; color:#666; }
.application-setup-steps li > span:first-child { display:inline-flex; width:18px; height:18px; align-items:center; justify-content:center; border:1px solid #414141; border-radius:50%; font-size:11px; }
.application-setup-steps .done { color:#aaa; }.application-setup-steps .current { color:#eee; }.application-setup-steps .current > span:first-child { border-color:#35acd7; color:#35acd7; }
.application-step-state { position:absolute; width:1px; height:1px; padding:0; margin:-1px; overflow:hidden; clip:rect(0,0,0,0); white-space:nowrap; }
@media(max-width:480px) { .application-setup-steps { gap:6px; font-size:11px; }.application-setup-steps li { gap:4px; } }
</style>
