import { describe, expect, it } from "vitest";
import { readHomeStatistics } from "./home-statistics";
import type { GroupStatistics } from "./api/types/data-contracts";

const zero: GroupStatistics = {
  totalItemPrice: 0,
  totalItems: 0,
  totalLocations: 0,
  totalTags: 0,
  totalUsers: 1,
  totalWithWarranty: 0,
};

describe("Home collection statistics", () => {
  it("preserves a legitimate all-zero collection", () => {
    expect(readHomeStatistics({ error: false, data: zero })).toEqual(zero);
  });

  it("preserves API value and count semantics without rounding or substituting quantities", () => {
    const data = { ...zero, totalItemPrice: 1234.567, totalItems: 12, totalLocations: 3, totalTags: 7 };
    expect(readHomeStatistics({ error: false, data })).toBe(data);
  });

  it("does not turn an HTTP failure into zero totals, even with cached-looking data", () => {
    expect(() => readHomeStatistics({ error: true, data: zero })).toThrow("Unable to load");
  });

  it.each([null, undefined, {}, { ...zero, totalItems: undefined }, { ...zero, totalItemPrice: NaN }])(
    "rejects missing or malformed statistics: %j",
    data => {
      expect(() => readHomeStatistics({ error: false, data: data as GroupStatistics })).toThrow("Unable to load");
    }
  );
});
