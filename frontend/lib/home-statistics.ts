import type { GroupStatistics } from "./api/types/data-contracts";

// Requests returns HTTP failures as responses rather than throwing. Do not
// turn a failed or incomplete response into a successful zero-valued collection.
export function readHomeStatistics(response: { error: boolean; data: GroupStatistics }): GroupStatistics {
  const data = response.data;
  if (
    response.error ||
    !data ||
    ![data.totalItemPrice, data.totalItems, data.totalLocations, data.totalTags].every(
      value => typeof value === "number" && Number.isFinite(value)
    )
  ) {
    throw new Error("Unable to load collection statistics");
  }
  return data;
}
