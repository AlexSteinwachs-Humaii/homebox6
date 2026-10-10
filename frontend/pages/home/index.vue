<script setup lang="ts">
  import { useI18n } from "vue-i18n";
  import { statCardData } from "./statistics";
  import { itemsTable } from "./table";
  import { useTagStore } from "~/stores/tags";
  import { useLocationStore } from "~~/stores/locations";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import StatisticCard from "~/components/Home/StatisticCard.vue";
  import { Button } from "~/components/ui/button";
  import ItemCard from "~/components/Item/Card.vue";
  import LocationCard from "~/components/Location/Card.vue";
  import TagChip from "~/components/Tag/Chip.vue";
  import Table from "~/components/Item/View/Table.vue";

  const { t } = useI18n();

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "One World Direct | " + t("menu.home"),
  });

  const api = useUserApi();
  const breakpoints = useBreakpoints();

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
        <p class="mt-2 text-sm text-muted-foreground">{{ $t("home.introduction_description") }}</p>
      </section>

      <section aria-labelledby="home-statistics">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 id="home-statistics" class="text-base font-bold">{{ $t("home.quick_statistics") }}</h2>
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

      <section>
        <Subtitle> {{ $t("home.recently_added") }} </Subtitle>

        <p v-if="itemTable.items.length === 0" class="ml-2 text-sm">{{ $t("items.no_results") }}</p>
        <BaseCard v-else-if="breakpoints.lg">
          <Table :items="itemTable.items" />
        </BaseCard>
        <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <ItemCard v-for="item in itemTable.items" :key="item.id" :item="item" />
        </div>
      </section>

      <section>
        <Subtitle> {{ $t("home.storage_locations") }} </Subtitle>
        <p v-if="locations.length === 0" class="ml-2 text-sm">{{ $t("locations.no_results") }}</p>
        <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          <LocationCard v-for="location in locations" :key="location.id" :location="location" />
        </div>
      </section>

      <section>
        <Subtitle> {{ $t("home.tags") }} </Subtitle>
        <p v-if="tags.length === 0" class="ml-2 text-sm">{{ $t("tags.no_results") }}</p>
        <div v-else class="flex flex-wrap gap-4">
          <TagChip v-for="tag in tags" :key="tag.id" size="lg" :tag="tag" class="shadow-md" />
        </div>
      </section>
    </BaseContainer>
  </div>
</template>
