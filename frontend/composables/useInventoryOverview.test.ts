import { describe, expect, it, vi } from "vitest";
import { effectScope, nextTick, ref } from "vue";
import { useInventoryOverview } from "./useInventoryOverview";
import type { UserClient } from "../lib/api/user";

const response = (data: unknown, error = false) => ({ data, error });
const rows = (name: string, total = 45) => ({
  items: [{ id: name, name }],
  total,
});
const flush = async () => {
  await nextTick();
  await Promise.resolve();
  await Promise.resolve();
};
function setup() {
  const getAll = vi.fn(async (query: { isLocation?: boolean }) =>
    response(query.isLocation ? { items: [] } : rows("first"))
  );
  const group = vi.fn(async () => response({ totalItems: 45 }));
  const client = () => ({ items: { getAll }, stats: { group } }) as unknown as UserClient;
  const collection = ref<string | null>("collection-a");
  const scope = effectScope();
  const overview = scope.run(() => useInventoryOverview(collection, client))!;
  return { overview, collection, scope, getAll, group };
}

describe("inventory overview", () => {
  it("sends search and descendant location filters before pagination, resets page, and keeps totals", async () => {
    const { overview: o, scope, getAll, group } = setup();
    await flush();
    o.page.value = 2;
    await flush();
    expect(getAll).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, pageSize: 20 }));
    o.search.value = "Drill";
    await flush();
    expect(o.page.value).toBe(1);
    expect(getAll).toHaveBeenLastCalledWith(expect.objectContaining({ q: "Drill", page: 1 }));
    o.location.value = "garage";
    await flush();
    expect(getAll).toHaveBeenLastCalledWith(
      expect.objectContaining({
        parentIds: ["garage"],
        includeDescendants: true,
        includeLocation: true,
      })
    );
    expect(group).toHaveBeenCalledTimes(1);
    expect(o.statistics.value?.totalItems).toBe(45);
    o.clearFilters();
    await flush();
    expect(o.filtered.value).toBe(false);
    scope.stop();
  });

  it("clears old data synchronously and discards responses from the previous collection", async () => {
    const { overview: o, collection, scope, getAll, group } = setup();
    await flush();
    let resolveRows!: (value: ReturnType<typeof response>) => void;
    let resolveStats!: (value: ReturnType<typeof response>) => void;
    getAll.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveRows = resolve;
        })
    );
    group.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveStats = resolve;
        })
    );
    o.search.value = "pending";
    void o.loadSummary();
    collection.value = "collection-b";
    expect(o.statistics.value).toBeUndefined();
    expect(o.result.value).toBeUndefined();
    expect(o.search.value).toBe("");
    expect(o.location.value).toBe("");
    expect(o.page.value).toBe(1);
    await flush();
    resolveRows(response(rows("secret-a")));
    resolveStats(response({ totalItems: 999 }));
    await flush();
    expect(o.result.value?.items[0]?.name).toBe("first");
    expect(o.statistics.value?.totalItems).toBe(45);
    scope.stop();
  });

  it("shows initial loading and summary errors with recovery", async () => {
    const { overview: o, scope, group } = setup();
    expect(o.loading.value).toBe(true);
    expect(o.summaryLoading.value).toBe(true);
    await flush();
    expect(o.loading.value).toBe(false);
    group.mockImplementationOnce(async () => response(null, true));
    await o.loadSummary();
    expect(o.summaryError.value).toBe(true);
    await o.loadSummary();
    expect(o.summaryError.value).toBe(false);
    scope.stop();
  });

  it("discards responses after leaving the dashboard", async () => {
    const { overview: o, scope, getAll } = setup();
    await flush();
    let resolve!: (value: ReturnType<typeof response>) => void;
    getAll.mockImplementationOnce(
      () =>
        new Promise(done => {
          resolve = done;
        })
    );
    o.search.value = "pending";
    scope.stop();
    resolve(response(rows("late")));
    await flush();
    expect(o.result.value).toBeUndefined();
  });

  it("ignores older filtered responses, exposes request errors, and supports retry", async () => {
    const { overview: o, scope, getAll } = setup();
    await flush();
    let resolve!: (value: ReturnType<typeof response>) => void;
    getAll.mockImplementationOnce(
      () =>
        new Promise(done => {
          resolve = done;
        })
    );
    o.search.value = "older";
    o.search.value = "newer";
    await flush();
    resolve(response(rows("older")));
    await flush();
    expect(o.result.value?.items[0]?.name).toBe("first");
    getAll.mockImplementationOnce(async () => response(null, true));
    await o.loadRows();
    expect(o.error.value).toBe(true);
    expect(o.result.value).toBeUndefined();
    await o.loadRows();
    expect(o.error.value).toBe(false);
    expect(o.result.value?.total).toBe(45);
    scope.stop();
  });
});
