import type { EntitySummary } from "./api/types/data-contracts";

/** The API orders createdAt descending; retain its ordering and count semantics. */
export function readRecentItems(response: { error: unknown; data?: { items?: EntitySummary[] } | null }) {
  if (response.error || !Array.isArray(response.data?.items)) {
    throw new Error("Unable to load recent inventory");
  }
  return response.data.items.slice(0, 5);
}

export function recentItemLocation(item: EntitySummary) {
  // Older responses can carry location; unified entities carry their parent instead.
  return (item as EntitySummary & { location?: { id: string; name: string } | null }).location || item.parent || null;
}

export function locationItemCount(location: EntitySummary | { itemCount?: number }) {
  return typeof location.itemCount === "number" && Number.isFinite(location.itemCount) ? location.itemCount : null;
}
