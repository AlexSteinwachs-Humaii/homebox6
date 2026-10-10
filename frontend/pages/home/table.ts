import type { UserClient } from "~~/lib/api/user";
import { readRecentItems } from "~/lib/home-inventory";

export function itemsTable(api: UserClient) {
  const preferences = useViewPreferences();
  const {
    data: items,
    refresh,
    status,
    error,
  } = useAsyncData(
    `home:recent:${preferences.value.collectionId ?? "default"}`,
    async () => {
      const response = await api.items.getAll({
        page: 1,
        pageSize: 5,
        orderBy: "createdAt",
      });
      return readRecentItems(response);
    },
    {
      deep: true,
      lazy: true,
    }
  );

  onServerEvent(ServerEvent.EntityMutation, () => {
    refresh();
  });

  return computed(() => {
    return {
      items: items.value || [],
      loading: status.value === "pending" || status.value === "idle",
      failed: !!error.value,
      refresh,
    };
  });
}
