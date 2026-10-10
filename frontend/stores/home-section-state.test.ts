import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useLocationStore } from "./locations";
import { useTagStore } from "./tags";

const getLocations = vi.fn();
const getTags = vi.fn();

beforeEach(() => {
  setActivePinia(createPinia());
  vi.stubGlobal("useUserApi", () => ({ items: { getLocations }, tags: { getAll: getTags } }));
  getLocations.mockReset();
  getTags.mockReset();
});

afterEach(() => vi.unstubAllGlobals());

for (const kind of ["locations", "tags"] as const) {
  describe(`Home ${kind} section status`, () => {
    function section() {
      if (kind === "locations") {
        const store = useLocationStore();
        return {
          load: () => store.refreshParents(),
          status: () => store.parentsStatus,
          data: () => store.parentLocations,
          request: getLocations,
        };
      }
      const store = useTagStore();
      return { load: () => store.refresh(), status: () => store.status, data: () => store.tags, request: getTags };
    }

    it("distinguishes loading from a legitimately empty result", async () => {
      const view = section();
      let finish!: (value: unknown) => void;
      view.request.mockImplementation(
        () =>
          new Promise(resolve => {
            finish = resolve;
          })
      );
      expect(view.status()).toBe("idle");
      const pending = view.load();
      expect(view.status()).toBe("pending");
      finish({ error: false, data: [] });
      await pending;
      expect(view.status()).toBe("success");
      expect(view.data()).toEqual([]);
      if (kind === "locations") expect(getLocations).toHaveBeenCalledWith({ filterChildren: true });
    });

    it("marks an API failure as error and permits retry", async () => {
      const view = section();
      view.request.mockResolvedValueOnce({ error: true }).mockResolvedValueOnce({ error: false, data: [] });
      await view.load();
      expect(view.status()).toBe("error");
      await view.load();
      expect(view.status()).toBe("success");
    });

    it("marks thrown network failures as errors", async () => {
      const view = section();
      view.request.mockRejectedValueOnce(new Error("Network failure"));
      await expect(view.load()).rejects.toThrow("Network failure");
      expect(view.status()).toBe("error");
    });

    it("does not treat malformed API data as an empty collection", async () => {
      const view = section();
      view.request.mockResolvedValueOnce({ error: false, data: null });
      await view.load();
      expect(view.status()).toBe("error");
    });
  });
}
