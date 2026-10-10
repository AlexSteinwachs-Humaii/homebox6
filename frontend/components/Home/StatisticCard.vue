<script setup lang="ts">
  import { Box, DollarSign, MapPin, Tags } from "lucide-vue-next";
  import Currency from "~/components/global/Currency.vue";

  defineProps<{
    title: string;
    value: number | null;
    type: "currency" | "number";
    description: string;
    kind: "value" | "items" | "locations" | "tags";
    loading: boolean;
    failed: boolean;
  }>();

  const icons = { value: DollarSign, items: Box, locations: MapPin, tags: Tags };
</script>

<template>
  <!-- eslint-disable-next-line tailwindcss/no-custom-classname -->
  <div class="home-statistic" :class="{ 'home-statistic-value': kind === 'value' }" :aria-busy="loading">
    <div class="flex items-center justify-between gap-3">
      <h3 class="text-xs font-normal text-muted-foreground">{{ title }}</h3>
      <span class="rounded-md bg-accent p-1.5 text-accent-foreground" aria-hidden="true">
        <component :is="icons[kind]" class="size-4" />
      </span>
    </div>
    <div class="mt-2 break-words text-3xl font-bold leading-none tracking-tight">
      <span v-if="loading" class="text-sm font-normal text-muted-foreground">{{ $t("global.loading") }}</span>
      <span v-else-if="failed || value === null" class="text-sm font-normal text-muted-foreground">
        {{ $t("home.statistics_unavailable") }}
      </span>
      <Currency v-else-if="type === 'currency'" :amount="value" />
      <span v-else>{{ value }}</span>
    </div>
    <p class="mt-1 text-xs text-muted-foreground">{{ description }}</p>
  </div>
</template>

<style scoped>
  .home-statistic {
    position: relative;
    overflow: hidden;
    min-width: 0;
    padding: 1.25rem 1.375rem;
    border: 1px solid hsl(var(--border));
    border-radius: var(--radius);
    background: hsl(var(--card));
    color: hsl(var(--card-foreground));
    box-shadow: var(--workspace-panel-shadow);
  }

  .home-statistic-value::before {
    position: absolute;
    inset: 0 0 auto;
    height: 3px;
    background: hsl(var(--primary));
    content: "";
  }
</style>
