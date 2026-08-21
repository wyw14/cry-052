<script setup lang="ts">
defineProps<{ tone: 'green' | 'amber' | 'red' | 'gray', label: string }>();
</script>
<template><span class="status" :class="`status-${tone}`"><i></i>{{ label }}</span></template>
