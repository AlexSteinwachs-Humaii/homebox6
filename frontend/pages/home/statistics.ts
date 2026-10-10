import { useI18n } from "vue-i18n";
import type { UserClient } from "~~/lib/api/user";
import { readHomeStatistics } from "~~/lib/home-statistics";

export function statCardData(api: UserClient) {
  const { t } = useI18n();
  const prefs = useViewPreferences();
  // Collection selection performs a full reload; the cache must still be
  // tenant-specific when returning to Home in the same session.
  const key = `home:statistics:${prefs.value.collectionId ?? "default"}`;
  const {
    data: statistics,
    status,
    error,
    refresh,
  } = useAsyncData(key, async () => readHomeStatistics(await api.stats.group()), { lazy: true });

  const cards = computed(() => [
    {
      id: "value" as const,
      label: t("home.total_value"),
      value: statistics.value?.totalItemPrice ?? null,
      type: "currency" as const,
      description: t("home.value_description"),
    },
    {
      id: "items" as const,
      label: t("home.total_items"),
      value: statistics.value?.totalItems ?? null,
      type: "number" as const,
      description: t("home.items_description"),
    },
    {
      id: "locations" as const,
      label: t("home.total_locations"),
      value: statistics.value?.totalLocations ?? null,
      type: "number" as const,
      description: t("home.locations_description"),
    },
    {
      id: "tags" as const,
      label: t("home.total_tags"),
      value: statistics.value?.totalTags ?? null,
      type: "number" as const,
      description: t("home.tags_description"),
    },
  ]);

  return { cards, status, error, refresh };
}
