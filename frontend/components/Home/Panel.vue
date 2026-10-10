<script setup lang="ts">
  defineProps<{
    title: string;
    action: string;
    to: string;
    description?: string;
  }>();
</script>

<template>
  <!-- eslint-disable-next-line tailwindcss/no-custom-classname -->
  <section class="home-panel min-w-0" :aria-label="title">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b px-5 py-4">
      <div class="min-w-0">
        <h2 class="text-base font-bold">{{ title }}</h2>
        <p v-if="description" class="mt-1 text-xs text-muted-foreground">
          {{ description }}
        </p>
      </div>
      <NuxtLink :to="to" class="text-sm font-semibold text-accent-foreground hover:underline">
        {{ action }} <span aria-hidden="true">→</span>
      </NuxtLink>
    </header>
    <slot />
  </section>
</template>

<style scoped>
  .home-panel {
    overflow: hidden;
    border: 1px solid hsl(var(--border));
    border-radius: var(--radius);
    background: hsl(var(--card));
    color: hsl(var(--card-foreground));
    box-shadow: var(--workspace-panel-shadow);
  }
</style>
