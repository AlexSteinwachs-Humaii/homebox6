<template>
  <Card>
    <NuxtLink v-if="home" :to="`/location/${location.id}`" class="block h-full rounded-lg p-3 hover:bg-accent/30">
      <MdiMapMarkerOutline class="mb-2 size-5 text-muted-foreground" aria-hidden="true" />
      <h3 class="break-words text-sm font-semibold">{{ location.name }}</h3>
      <p class="mt-1 text-xs text-muted-foreground">
        {{ homeCount === null ? $t("home.not_available") : $t("home.location_records", { count: homeCount }) }}
      </p>
    </NuxtLink>
    <NuxtLink v-else :to="`/location/${location.id}`" class="group/location-card transition duration-300">
      <div
        :class="{
          'p-4': !dense,
          'px-3 py-2': dense,
        }"
      >
        <h2 class="flex items-center justify-between gap-2">
          <div class="relative size-6">
            <div
              class="absolute inset-0 flex items-center justify-center transition-transform duration-300 group-hover/location-card:-rotate-90"
            >
              <MdiMapMarkerOutline class="size-6 group-hover/location-card:hidden" />
              <MdiArrowUp class="hidden size-6 group-hover/location-card:block" />
            </div>
          </div>
          <span class="mx-auto">
            {{ location.name }}
          </span>
          <Badge :class="{ 'opacity-0': !hasCount }">
            {{ count }}
          </Badge>
        </h2>
      </div>
    </NuxtLink>
  </Card>
</template>

<script lang="ts" setup>
  import type { EntityOut, EntitySummary } from "~~/lib/api/types/data-contracts";
  import MdiArrowUp from "~icons/mdi/arrow-down";
  import MdiMapMarkerOutline from "~icons/mdi/map-marker-outline";
  import { Card } from "@/components/ui/card";
  import { Badge } from "@/components/ui/badge";
  import { locationItemCount } from "~/lib/home-inventory";

  const props = defineProps({
    location: {
      type: Object as () => EntitySummary | EntityOut,
      required: true,
    },
    home: {
      type: Boolean,
      default: false,
    },
    dense: {
      type: Boolean,
      default: false,
    },
  });

  const homeCount = computed(() => locationItemCount(props.location as EntitySummary));
  const hasCount = computed(() => homeCount.value !== null);

  const count = computed(() => {
    return hasCount.value ? (props.location as EntitySummary).itemCount : undefined;
  });
</script>
