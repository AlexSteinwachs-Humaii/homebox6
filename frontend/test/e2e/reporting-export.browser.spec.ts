import { expect, test } from "@playwright/test";

// Story 3 covers the overview. CSV lifecycle/download assertions belong to story 4/5.
test("Reports preserves Home and shows collection totals with server-paginated filters", async ({ page }) => {
  await page.goto("/");
  await page.fill("input[type='text']", "demo@example.com");
  await page.fill("input[type='password']", "demodemo");
  await page.click("button[type='submit']");
  await expect(page).toHaveURL("/home");

  let fail = false;
  let empty = false;
  const queries: URLSearchParams[] = [];
  await page.route("**/api/v1/groups/statistics", route =>
    route.fulfill({
      json: {
        totalItemPrice: 4680,
        totalItems: 45,
        totalLocations: 6,
        totalTags: 8,
      },
    })
  );
  await page.route("**/api/v1/entities?*", async route => {
    const query = new URL(route.request().url()).searchParams;
    if (query.get("isLocation") === "true") {
      await route.fulfill({ json: { items: [{ id: "garage", name: "Garage" }], total: 1 } });
      return;
    }
    queries.push(query);
    if (fail) {
      await route.fulfill({ status: 500, json: { error: "test failure" } });
      return;
    }
    const noMatch = query.get("q") === "nothing";
    await route.fulfill({
      json: {
        items:
          empty || noMatch
            ? []
            : [
                {
                  id: "drill",
                  name: `Drill page ${query.get("page")}`,
                  assetId: "000-012",
                  quantity: 2,
                  parent: { id: "box", name: "Toolbox" },
                  location: { id: "garage", name: "Garage" },
                },
              ],
        total: empty || noMatch ? 0 : 45,
        page: Number(query.get("page")),
        pageSize: 20,
      },
    });
  });

  await page.getByRole("link", { name: "Reports", exact: true }).click();
  await expect(page).toHaveURL("/reports");
  await expect(page.getByRole("heading", { name: "Inventory overview" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Drill page 1" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Garage", exact: true })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Toolbox", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Next", exact: true }).click();
  await expect(page.getByRole("cell", { name: "Drill page 2" })).toBeVisible();
  await page.getByLabel("Search inventory").fill("Drill");
  await expect(page.getByRole("cell", { name: "Drill page 1" })).toBeVisible();
  await page.getByLabel("Location", { exact: true }).selectOption("garage");
  await expect.poll(() => queries.at(-1)?.get("parentIds")).toBe("garage");
  expect(queries.at(-1)?.get("q")).toBe("Drill");
  expect(queries.at(-1)?.get("page")).toBe("1");
  expect(queries.at(-1)?.get("includeDescendants")).toBe("true");
  await expect(page.getByText("45", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Export all data (CSV)" })).toBeVisible();
  await expect(page.getByText(/Search, location filters and pagination do not apply/)).toBeVisible();

  await page.getByLabel("Search inventory").fill("nothing");
  await expect(page.getByText(/No inventory matches these filters/)).toBeVisible();
  await expect(page.getByRole("button", { name: "Export all data (CSV)" })).toBeVisible();
  fail = true;
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page.getByText("Could not load inventory. Please retry.")).toBeVisible();
  fail = false;
  empty = true;
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await expect(page.getByText("This collection has no inventory items.")).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByLabel("Search inventory")).toBeVisible();
  await expect(page.getByRole("button", { name: "Export all data (CSV)" })).toBeVisible();
});

test("Reports requires authentication", async ({ page }) => {
  await page.goto("/reports");
  await expect(page).toHaveURL("/");
});
