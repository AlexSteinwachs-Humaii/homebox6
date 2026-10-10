<script setup lang="ts">
  // markSrc must be the original supplied artwork, never a traced concept mark.
  // Until it is supplied we show an explicitly typographic fallback, not a fake logo.
  withDefaults(defineProps<{ markSrc?: string; collapsed?: boolean }>(), {
    markSrc: undefined,
    collapsed: false,
  });
</script>

<template>
  <span class="workspace-brand" :class="{ 'workspace-brand-collapsed': collapsed }">
    <img v-if="markSrc" :src="markSrc" alt="" class="workspace-brand-mark" />
    <span v-else class="workspace-brand-mark workspace-brand-pending" aria-hidden="true">OWD</span>
    <span class="sr-only">One World Direct — {{ $t("workspace.brand_descriptor") }}</span>
    <span class="workspace-brand-copy" aria-hidden="true">
      <span class="workspace-brand-name">ONE WORLD<br />DIRECT</span>
      <span class="workspace-brand-description block">{{ $t("workspace.brand_descriptor") }}</span>
    </span>
  </span>
</template>
