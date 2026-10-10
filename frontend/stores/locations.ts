import { defineStore } from "pinia";
import type { ItemsApi } from "~~/lib/api/classes/items";
import type { EntitySummary, TreeItem } from "~~/lib/api/types/data-contracts";

export const useLocationStore = defineStore("locations", {
  state: () => ({
    parents: null as EntitySummary[] | null,
    parentsStatus: "idle" as "idle" | "pending" | "success" | "error",
    Locations: null as EntitySummary[] | null,
    client: useUserApi(),
    tree: null as TreeItem[] | null,
    refreshLocationsPromise: null as Promise<void> | null,
  }),
  getters: {
    /**
     * locations represents the locations that are currently in the store. The store is
     * synched with the server by intercepting the API calls and updating on the
     * response
     */
    parentLocations(state): EntitySummary[] {
      // The default layout loads and refreshes parents. Reading this getter must
      // not start duplicate requests or repeatedly retry a failed section.
      return state.parents ?? [];
    },
    allLocations(state): EntitySummary[] {
      return state.Locations ?? [];
    },
  },
  actions: {
    async ensureLocationsFetched() {
      if (this.Locations !== null) {
        return;
      }

      if (this.refreshLocationsPromise === null) {
        this.refreshLocationsPromise = this.refreshChildren().then(() => {});
      }
      await this.refreshLocationsPromise;
    },
    async refreshParents(): ReturnType<ItemsApi["getLocations"]> {
      this.parentsStatus = "pending";
      try {
        const result = await this.client.items.getLocations({
          filterChildren: true,
        });
        if (result.error || !Array.isArray(result.data)) {
          this.parentsStatus = "error";
          return result;
        }

        this.parents = result.data;
        this.parentsStatus = "success";
        return result;
      } catch (error) {
        this.parentsStatus = "error";
        throw error;
      }
    },
    async refreshChildren(): ReturnType<ItemsApi["getLocations"]> {
      const result = await this.client.items.getLocations({
        filterChildren: false,
      });
      if (result.error) {
        return result;
      }

      this.Locations = result.data;
      return result;
    },
    async refreshTree(): ReturnType<ItemsApi["getTree"]> {
      const result = await this.client.items.getTree();
      if (result.error) {
        return result;
      }

      this.tree = result.data;
      return result;
    },
  },
});
