<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { statCardData } from "./statistics";
  import { itemsTable } from "./table";
  import { useTagStore } from "~/stores/tags";
  import { useLocationStore } from "~~/stores/locations";
  import BaseContainer from "@/components/Base/Container.vue";
  import Panel from "~/components/Home/Panel.vue";
  import RecentItems from "~/components/Home/RecentItems.vue";
  import StatisticCard from "~/components/Home/StatisticCard.vue";
  import { Button } from "~/components/ui/button";
  import LocationCard from "~/components/Location/Card.vue";
  import TagChip from "~/components/Tag/Chip.vue";

  const { t } = useI18n();

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "One World Direct | " + t("menu.home"),
  });

  const api = useUserApi();

  const locationStore = useLocationStore();
  const locations = computed(() => locationStore.parentLocations);

  const tagsStore = useTagStore();
  const tags = computed(() => tagsStore.tags);

  const itemTable = itemsTable(api);
  const { selectedCollection } = useCollections();
  const {
    cards: stats,
    status: statisticsStatus,
    error: statisticsError,
    refresh: refreshStatistics,
  } = statCardData(api);
  const statisticsLoading = computed(() => statisticsStatus.value === "pending" || statisticsStatus.value === "idle");
  const statisticsFailed = computed(() => !!statisticsError.value);
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6 lg:my-6 lg:px-9">
      <section aria-labelledby="home-introduction">
        <p class="mb-1.5 text-[11px] font-bold uppercase tracking-[0.18em] text-accent-foreground">
          {{ $t("home.inventory_eyebrow") }}
        </p>
        <h1 id="home-introduction" class="text-3xl font-bold leading-tight tracking-tight">
          {{ $t("home.introduction_title") }}
        </h1>
        <p class="mt-2 text-sm text-muted-foreground">
          {{ $t("home.introduction_description") }}
        </p>
      </section>

      <section aria-labelledby="home-statistics">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 id="home-statistics" class="text-base font-bold">
            {{ $t("home.quick_statistics") }}
          </h2>
          <p v-if="selectedCollection" class="break-words text-xs text-muted-foreground">
            {{ $t("home.collection_context", { name: selectedCollection.name }) }}
          </p>
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatisticCard
            v-for="stat in stats"
            :key="stat.id"
            :kind="stat.id"
            :title="stat.label"
            :value="stat.value"
            :type="stat.type"
            :description="stat.description"
            :loading="statisticsLoading"
            :failed="statisticsFailed"
          />
        </div>
        <div v-if="statisticsFailed" role="alert" class="mt-3 flex flex-wrap items-center gap-3 text-sm">
          <p>{{ $t("home.statistics_error") }}</p>
          <Button variant="outline" size="sm" @click="refreshStatistics()">{{ $t("home.statistics_retry") }}</Button>
        </div>
      </section>

      <Panel
        :title="$t('home.recently_added')"
        :description="$t('home.recent_description')"
        :action="$t('home.search_inventory')"
        to="/items"
        :aria-busy="itemTable.loading"
      >
        <p v-if="itemTable.loading" role="status" class="p-5 text-sm text-muted-foreground">
          {{ $t("global.loading") }}
        </p>
        <div v-else-if="itemTable.failed" role="alert" class="flex flex-wrap items-center gap-3 p-5 text-sm">
          <p>{{ $t("home.recent_error") }}</p>
          <Button variant="outline" size="sm" @click="itemTable.refresh()">{{ $t("home.retry") }}</Button>
        </div>
        <p v-else-if="itemTable.items.length === 0" class="p-5 text-sm text-muted-foreground">
          {{ $t("items.no_results") }}
        </p>
        <RecentItems v-else :items="itemTable.items" />
      </Panel>

      <div class="grid grid-cols-1 gap-5 lg:grid-cols-5">
        <Panel
          :title="$t('home.storage_locations')"
          :action="$t('home.view_all')"
          to="/locations"
          class="lg:col-span-3"
          :aria-busy="locationStore.parentsStatus === 'pending' || locationStore.parentsStatus === 'idle'"
        >
          <div class="p-4">
            <p
              v-if="locationStore.parentsStatus === 'pending' || locationStore.parentsStatus === 'idle'"
              role="status"
              class="text-sm text-muted-foreground"
            >
              {{ $t("global.loading") }}
            </p>
            <div
              v-else-if="locationStore.parentsStatus === 'error'"
              role="alert"
              class="flex flex-wrap items-center gap-3 text-sm"
            >
              <p>{{ $t("home.locations_error") }}</p>
              <Button variant="outline" size="sm" @click="locationStore.refreshParents()">{{
                $t("home.retry")
              }}</Button>
            </div>
            <p v-else-if="locations.length === 0" class="text-sm text-muted-foreground">
              {{ $t("locations.no_results") }}
            </p>
            <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <LocationCard v-for="location in locations" :key="location.id" :location="location" home />
            </div>
          </div>
        </Panel>

        <Panel
          :title="$t('home.tags')"
          :action="$t('home.manage')"
          to="/tags"
          class="lg:col-span-2"
          :aria-busy="tagsStore.status === 'pending' || tagsStore.status === 'idle'"
        >
          <div class="p-5">
            <p
              v-if="tagsStore.status === 'pending' || tagsStore.status === 'idle'"
              role="status"
              class="text-sm text-muted-foreground"
            >
              {{ $t("global.loading") }}
            </p>
            <div
              v-else-if="tagsStore.status === 'error'"
              role="alert"
              class="flex flex-wrap items-center gap-3 text-sm"
            >
              <p>{{ $t("home.tags_error") }}</p>
              <Button variant="outline" size="sm" @click="tagsStore.refresh()">{{ $t("home.retry") }}</Button>
            </div>
            <p v-else-if="tags.length === 0" class="text-sm text-muted-foreground">
              {{ $t("tags.no_results") }}
            </p>
            <div v-else class="flex flex-wrap gap-2">
              <TagChip v-for="tag in tags" :key="tag.id" size="sm" :tag="tag" home />
            </div>
          </div>
        </Panel>
      </div>
    </BaseContainer>
  </div>
</template>
