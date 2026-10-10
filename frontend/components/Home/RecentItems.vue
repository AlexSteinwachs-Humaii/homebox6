<script setup lang="ts">
  import type { EntitySummary } from "~/lib/api/types/data-contracts";
  import { recentItemLocation } from "~/lib/home-inventory";

  defineProps<{ items: EntitySummary[] }>();
</script>

<template>
  <div class="overflow-x-auto" tabindex="0" role="region" :aria-label="$t('home.recently_added')">
    <p class="px-5 py-2 text-xs text-muted-foreground sm:hidden">{{ $t("home.scroll_columns") }}</p>
    <table class="w-full min-w-[640px] table-fixed text-left text-sm">
      <caption class="sr-only">
        {{
          $t("home.recently_added")
        }}
      </caption>
      <colgroup>
        <col class="w-[41%]" />
        <col class="w-[17%]" />
        <col class="w-[26%]" />
        <col class="w-[16%]" />
      </colgroup>
      <thead class="border-y bg-muted/30 text-xs uppercase tracking-wide text-muted-foreground">
        <tr>
          <th scope="col" class="px-5 py-3">{{ $t("items.name") }}</th>
          <th scope="col" class="px-5 py-3">{{ $t("items.asset_id") }}</th>
          <th scope="col" class="px-5 py-3">{{ $t("items.location") }}</th>
          <th scope="col" class="px-5 py-3 text-right">
            {{ $t("items.quantity") }}
          </th>
        </tr>
      </thead>
      <tbody class="divide-y">
        <tr v-for="item in items" :key="item.id">
          <td class="min-w-40 px-5 py-4">
            <NuxtLink :to="`/item/${item.id}`" class="break-words font-semibold hover:underline">
              {{ item.name }}
            </NuxtLink>
            <p v-if="item.entityType?.name" class="mt-1 text-xs text-muted-foreground">
              {{ item.entityType.name }}
            </p>
          </td>
          <td class="break-words px-5 py-4 font-mono text-xs text-muted-foreground">
            {{ item.assetId || $t("home.not_available") }}
          </td>
          <td class="min-w-32 px-5 py-4 text-muted-foreground">
            <NuxtLink
              v-if="recentItemLocation(item)"
              :to="`/location/${recentItemLocation(item)!.id}`"
              class="break-words hover:underline"
              >{{ recentItemLocation(item)!.name }}</NuxtLink
            >
            <span v-else>{{ $t("home.no_location") }}</span>
          </td>
          <td class="px-5 py-4 text-right">
            {{ item.quantity ?? $t("home.not_available") }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
