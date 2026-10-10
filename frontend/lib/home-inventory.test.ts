import { describe, expect, it } from "vitest";
import type { EntitySummary } from "./api/types/data-contracts";
import { locationItemCount, readRecentItems, recentItemLocation } from "./home-inventory";

const record = (fields: Partial<EntitySummary> = {}): EntitySummary => ({
  archived: false,
  assetId: "000-123",
  createdAt: "2026-10-01T12:00:00Z",
  description: "",
  id: "real-record",
  insured: false,
  itemCount: 0,
  name: "用户 inventory",
  purchasePrice: 0,
  quantity: 0,
  soldDate: "",
  tags: [],
  updatedAt: "",
  ...fields,
});

describe("Home recent inventory", () => {
  it("keeps API newest-first order and limits the section to five records", () => {
    const items = Array.from({ length: 7 }, (_, index) => record({ id: String(index), quantity: index }));
    const result = readRecentItems({ error: false, data: { items } });
    expect(result).toEqual(items.slice(0, 5));
    expect(items).toHaveLength(7);
    expect(result[0]).toBe(items[0]);
  });

  it("preserves real names, asset IDs, zero quantity and fewer than five records", () => {
    const item = record();
    expect(readRecentItems({ error: false, data: { items: [item] } })).toEqual([item]);
  });

  it("accepts a successful empty collection", () => {
    expect(readRecentItems({ error: false, data: { items: [] } })).toEqual([]);
  });

  it.each([
    { error: true, data: { items: [record()] } },
    { error: false, data: null },
    { error: false, data: {} },
  ])("does not present failed or missing data as an empty collection", response => {
    expect(() => readRecentItems(response)).toThrow("Unable to load recent inventory");
  });

  it("uses the actual unified entity parent", () => {
    const parent = record({ id: "parent", name: "Real location" });
    expect(recentItemLocation(record({ parent }))).toBe(parent);
  });

  it("supports legacy location data before parent", () => {
    const location = { id: "legacy-location", name: "Actual older location" };
    const item = { ...record({ parent: record() }), location };
    expect(recentItemLocation(item)).toBe(location);
  });

  it("does not invent a location when the record has none", () => {
    expect(recentItemLocation(record())).toBeNull();
  });
});

describe("Home location record counts", () => {
  it.each([0, 12])("retains the API count %i", itemCount => {
    expect(locationItemCount({ itemCount })).toBe(itemCount);
  });

  it.each([{}, { itemCount: NaN }])("distinguishes absent counts from zero", location => {
    expect(locationItemCount(location)).toBeNull();
  });
});
