import { expect, test } from "@playwright/test";

// Presentation checks isolate the current-collection statistics response. The
// full dashboard's live API/mutation/collection-switch walk is separate.
test("Home introduction and statistics distinguish pending, zero and failure", async ({ page, context, baseURL }) => {
  await page.setViewportSize({ width: 1440, height: 1160 });
  await context.addCookies([{ name: "hb.auth.session", value: "true", url: baseURL ?? "http://localhost:3000" }]);
  await page.addInitScript(() => {
    localStorage.setItem("homebox/preferences/location", JSON.stringify({ collectionId: "test-collection" }));
  });

  let finishStatistics: (() => void) | undefined;
  let mode: "pending" | "failed" | "ready" = "pending";
  const pending = new Promise<void>(resolve => {
    finishStatistics = resolve;
  });
  await page.route("**/api/v1/**", async route => {
    const path = new URL(route.request().url()).pathname;
    let data: unknown = [];
    if (path === "/api/v1/status") {
      data = { build: { version: "test", commit: "test" }, latest: { version: "test" } };
    } else if (path === "/api/v1/users/self") {
      data = {
        item: {
          id: "test-user",
          name: "Statistics Tester",
          email: "test@example.com",
          defaultGroupId: "test-collection",
        },
      };
    } else if (path === "/api/v1/users/self/settings") {
      data = { item: {} };
    } else if (path === "/api/v1/groups/all") {
      data = [{ id: "test-collection", name: "My collection", currency: "EUR" }];
    } else if (path === "/api/v1/groups") {
      data = { id: "test-collection", name: "My collection", currency: "EUR" };
    } else if (path === "/api/v1/groups/statistics") {
      if (mode === "pending") await pending;
      if (mode === "failed") {
        await route.fulfill({ status: 500, json: { error: "Test failure" } });
        return;
      }
      data = { totalItemPrice: 0, totalItems: 0, totalLocations: 0, totalTags: 0, totalUsers: 1, totalWithWarranty: 0 };
    } else if (path === "/api/v1/entities") {
      data = { items: [], total: 0 };
    }
    await route.fulfill({ json: data });
  });

  await page.goto("/home");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("A clear view of your inventory.");
  await expect(page.getByText("My collection collection", { exact: true })).toBeVisible();
  const cards = page.locator(".home-statistic");
  await expect(cards).toHaveCount(4);
  await expect(cards.first()).toContainText("Loading");
  await expect(cards.first()).not.toContainText("€0.00");
  const bounds = await cards.evaluateAll(elements => elements.map(element => element.getBoundingClientRect().top));
  expect(new Set(bounds).size).toBe(1);

  mode = "failed";
  finishStatistics?.();
  await expect(page.getByRole("alert")).toContainText("Collection statistics could not be loaded");
  await expect(cards.first()).toContainText("Unavailable");
  await expect(cards.first()).not.toContainText("€0.00");

  mode = "ready";
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(cards.first()).toContainText("€0.00");
  for (let i = 1; i < 4; i++) await expect(cards.nth(i)).toContainText("0");
  await expect(page.getByText(/Illustrative data|Desktop design concept|Warehouse inventory/)).toHaveCount(0);
  await page.screenshot({ path: test.info().outputPath("home-statistics-desktop.png"), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(cards.first()).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
