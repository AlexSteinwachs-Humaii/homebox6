<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import BaseContainer from "~/components/Base/Container.vue";
  import StatCard from "~/components/global/StatCard/StatCard.vue";
  import { Button } from "~/components/ui/button";
  import { Input } from "~/components/ui/input";
  import { Card } from "~/components/ui/card";

  definePageMeta({ middleware: ["auth"] });
  const { t } = useI18n();
  useHead({ title: "HomeBox | " + t("reports.overview.title") });
  const prefs = useViewPreferences();
  const collection = computed(() => prefs.value.collectionId);
  const {
    search,
    location,
    page,
    pageSize,
    pages,
    filtered,
    statistics,
    locations,
    result,
    loading,
    summaryLoading,
    error,
    summaryError,
    loadRows,
    loadSummary,
    clearFilters,
  } = useInventoryOverview(collection, useUserApi);
  const stats = computed(() => [
    {
      label: t("home.total_value"),
      value: statistics.value?.totalItemPrice ?? 0,
      type: "currency" as const,
    },
    {
      label: t("home.total_items"),
      value: statistics.value?.totalItems ?? 0,
      type: "number" as const,
    },
    {
      label: t("home.total_locations"),
      value: statistics.value?.totalLocations ?? 0,
      type: "number" as const,
    },
    {
      label: t("home.total_tags"),
      value: statistics.value?.totalTags ?? 0,
      type: "number" as const,
    },
  ]);
</script>

<template>
  <BaseContainer class="flex flex-col gap-6">
    <header class="flex flex-col justify-between gap-4 sm:flex-row">
      <div>
        <h1 class="text-2xl font-semibold">
          {{ t("reports.overview.title") }}
        </h1>
        <p class="text-muted-foreground">{{ t("reports.overview.intro") }}</p>
      </div>
      <div class="max-w-md sm:text-right">
        <!-- Background start/status/download controls are implemented in the next story. -->
        <Button disabled aria-describedby="export-scope export-pending">{{ t("reports.overview.export") }}</Button>
        <p id="export-scope" class="mt-2 text-sm text-muted-foreground">
          {{ t("reports.overview.export_scope") }}
        </p>
        <p id="export-pending" class="text-sm text-muted-foreground">
          {{ t("reports.overview.export_pending") }}
        </p>
      </div>
    </header>

    <section :aria-label="t('home.quick_statistics')" :aria-busy="summaryLoading">
      <p v-if="summaryLoading" role="status">
        {{ t("reports.overview.loading_totals") }}
      </p>
      <div v-else-if="summaryError" role="alert" class="flex items-center gap-3">
        <p>{{ t("reports.overview.summary_error") }}</p>
        <Button variant="outline" @click="loadSummary">{{ t("reports.overview.retry") }}</Button>
      </div>
      <div v-else-if="statistics" class="grid grid-cols-2 gap-2 md:grid-cols-4 md:gap-6">
        <StatCard v-for="stat in stats" :key="stat.label" :title="stat.label" :value="stat.value" :type="stat.type" />
      </div>
    </section>

    <Card class="overflow-hidden">
      <div class="space-y-4 p-6">
        <h2 class="text-lg font-medium">{{ t("reports.overview.data") }}</h2>
        <p class="text-sm text-muted-foreground">
          {{ t("reports.overview.filter_scope") }}
        </p>
        <div class="flex flex-col items-start gap-4 sm:flex-row sm:items-end">
          <div class="w-full sm:max-w-sm">
            <label for="inventory-search" class="mb-1 block text-sm">{{ t("reports.overview.search") }}</label>
            <Input id="inventory-search" v-model="search" type="search" />
          </div>
          <div class="w-full sm:max-w-xs">
            <label for="inventory-location" class="mb-1 block text-sm">{{ t("reports.overview.location") }}</label>
            <select
              id="inventory-location"
              v-model="location"
              :disabled="summaryLoading || summaryError"
              class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline focus-visible:outline-2"
            >
              <option value="">
                {{ t("reports.overview.all_locations") }}
              </option>
              <option v-for="loc in locations" :key="loc.id" :value="loc.id">
                {{ loc.name }}
              </option>
            </select>
          </div>
          <Button variant="outline" :disabled="!filtered" @click="clearFilters">{{
            t("reports.overview.clear")
          }}</Button>
        </div>
      </div>
      <div :aria-busy="loading" aria-live="polite">
        <p v-if="loading" role="status" class="p-6">
          {{ t("reports.overview.loading") }}
        </p>
        <div v-else-if="error" role="alert" class="flex items-center gap-3 p-6">
          <p>{{ t("reports.overview.error") }}</p>
          <Button variant="outline" @click="loadRows">{{ t("reports.overview.retry") }}</Button>
        </div>
        <p v-else-if="!result?.items.length" class="p-6">
          {{ t(filtered ? "reports.overview.no_match" : "reports.overview.empty") }}
        </p>
        <div v-else class="overflow-x-auto" tabindex="0" role="region" :aria-label="t('reports.overview.data')">
          <table class="w-full min-w-[640px] text-left text-sm">
            <caption class="sr-only">
              {{
                t("reports.overview.data")
              }}
            </caption>
            <thead class="border-b">
              <tr>
                <th scope="col" class="px-6 py-3">
                  {{ t("reports.overview.name") }}
                </th>
                <th scope="col" class="px-6 py-3">
                  {{ t("reports.overview.asset_id") }}
                </th>
                <th scope="col" class="px-6 py-3">
                  {{ t("reports.overview.location") }}
                </th>
                <th scope="col" class="px-6 py-3">
                  {{ t("reports.overview.quantity") }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in result.items" :key="item.id" class="border-b">
                <td class="px-6 py-4">
                  <NuxtLink :to="`/item/${item.id}`" class="underline">{{ item.name }}</NuxtLink>
                </td>
                <td class="whitespace-nowrap px-6 py-4">{{ item.assetId }}</td>
                <td class="px-6 py-4">
                  {{ item.location?.name ?? t("reports.overview.no_location") }}
                </td>
                <td class="px-6 py-4">{{ item.quantity }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      <footer class="flex flex-wrap items-center justify-between gap-3 p-6">
        <p v-if="result" class="text-sm">
          {{
            t("reports.overview.range", {
              start: result.total ? (page - 1) * pageSize + 1 : 0,
              end: Math.min(page * pageSize, result.total),
              total: result.total,
            })
          }}
        </p>
        <nav :aria-label="t('reports.overview.pagination')" class="flex items-center gap-2">
          <span class="text-sm">{{ t("reports.overview.page", { page, pages }) }}</span>
          <Button variant="outline" :disabled="loading || error || page <= 1" @click="page--">{{
            t("reports.overview.previous")
          }}</Button>
          <Button variant="outline" :disabled="loading || error || page >= pages" @click="page++">{{
            t("reports.overview.next")
          }}</Button>
        </nav>
      </footer>
    </Card>
  </BaseContainer>
</template>
