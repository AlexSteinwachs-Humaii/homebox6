import { computed, onScopeDispose, ref, watch } from "vue";
import type { Ref } from "vue";
import type { UserClient } from "~~/lib/api/user";
import type { EntityListResult, EntitySummary, GroupStatistics } from "~~/lib/api/types/data-contracts";

// Each collection gets a fresh API client (X-Tenant is captured at construction).
// Generations discard pending responses on filter/collection changes or disposal.
export function useInventoryOverview(collection: Ref<string | null | undefined>, client: () => UserClient) {
  const search = ref("");
  const location = ref("");
  const page = ref(1);
  const pageSize = 20;
  const statistics = ref<GroupStatistics>();
  const locations = ref<EntitySummary[]>([]);
  const result = ref<EntityListResult>();
  const loading = ref(false);
  const summaryLoading = ref(false);
  const error = ref(false);
  const summaryError = ref(false);
  let generation = 0;
  let tableRequest = 0;
  let summaryRequest = 0;
  let resetting = false;
  let api: UserClient;

  async function loadRows() {
    const scope = generation;
    const request = ++tableRequest;
    result.value = undefined;
    error.value = false;
    loading.value = true;
    try {
      const response = await api.items.getAll({
        q: search.value.trim(),
        parentIds: location.value ? [location.value] : [],
        includeDescendants: true,
        includeLocation: true,
        page: page.value,
        pageSize,
      });
      if (scope !== generation || request !== tableRequest) return;
      if (response.error) throw new Error("Inventory request failed");
      const lastPage = Math.max(1, Math.ceil(response.data.total / pageSize));
      if (page.value > lastPage) {
        page.value = lastPage;
        return;
      }
      result.value = response.data;
    } catch {
      if (scope === generation && request === tableRequest) error.value = true;
    } finally {
      if (scope === generation && request === tableRequest) loading.value = false;
    }
  }

  async function loadSummary() {
    const scope = generation;
    const request = ++summaryRequest;
    summaryLoading.value = true;
    summaryError.value = false;
    try {
      const [stats, locs] = await Promise.all([
        api.stats.group(),
        api.items.getAll({ isLocation: true, page: -1, pageSize: -1 }),
      ]);
      if (scope !== generation || request !== summaryRequest) return;
      if (stats.error || locs.error) throw new Error("Summary request failed");
      statistics.value = stats.data;
      locations.value = locs.data.items;
    } catch {
      if (scope === generation && request === summaryRequest) summaryError.value = true;
    } finally {
      if (scope === generation && request === summaryRequest) summaryLoading.value = false;
    }
  }

  watch(
    [search, location],
    () => {
      if (resetting) return;
      if (page.value !== 1) page.value = 1;
      else void loadRows();
    },
    { flush: "sync" }
  );
  watch(
    page,
    () => {
      if (!resetting) void loadRows();
    },
    { flush: "sync" }
  );
  watch(
    collection,
    () => {
      ++generation;
      ++tableRequest;
      ++summaryRequest;
      resetting = true;
      statistics.value = undefined;
      locations.value = [];
      result.value = undefined;
      search.value = "";
      location.value = "";
      page.value = 1;
      error.value = false;
      summaryError.value = false;
      resetting = false;
      api = client();
      void loadSummary();
      void loadRows();
    },
    { immediate: true, flush: "sync" }
  );
  onScopeDispose(() => {
    ++generation;
  });

  const pages = computed(() => Math.max(1, Math.ceil((result.value?.total ?? 0) / pageSize)));
  const filtered = computed(() => !!search.value.trim() || !!location.value);
  function clearFilters() {
    resetting = true;
    search.value = "";
    location.value = "";
    page.value = 1;
    resetting = false;
    void loadRows();
  }
  return {
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
  };
}
