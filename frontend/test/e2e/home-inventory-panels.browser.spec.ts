import { expect, test } from "@playwright/test";

// Home presentation only. Live collection switching/mutation coverage belongs to
// the dashboard refresh story; none of these fixtures are production data.
test("recent inventory and shortcuts retain actual API values and destinations", async ({ page, context, baseURL }) => {
  await page.setViewportSize({ width: 1440, height: 1160 });
  await context.addCookies([{ name: "hb.auth.session", value: "true", url: baseURL ?? "http://localhost:3000" }]);
  await page.addInitScript(() => {
    localStorage.setItem("homebox/preferences/location", JSON.stringify({ collectionId: "panel-test" }));
  });

  const location = { id: "real-location", name: "用户 Storage", itemCount: 0 };
  const longName = "Long inventory name " + "X".repeat(100);
  const items = Array.from({ length: 7 }, (_, index) => ({
    id: `record-${index}`,
    name: index === 0 ? longName : `Inventory ${index}`,
    assetId: `000-${107 - index}`,
    quantity: index,
    createdAt: `2026-10-0${7 - index}T12:00:00Z`,
    parent: index === 0 ? null : location,
  }));
  let mode: "ready" | "failed" | "empty" = "ready";
  const requests: URL[] = [];
  let finishSections!: () => void;
  const sectionsPending = new Promise<void>(resolve => {
    finishSections = resolve;
  });
  await page.route("**/api/v1/**", async route => {
    const url = new URL(route.request().url());
    requests.push(url);
    const path = url.pathname;
    let data: unknown = [];
    if (path === "/api/v1/status") data = { build: { version: "test" }, latest: { version: "test" } };
    else if (path === "/api/v1/users/self")
      data = { item: { id: "tester", name: "Panel Tester", email: "test@example.com", defaultGroupId: "panel-test" } };
    else if (path === "/api/v1/users/self/settings") data = { item: {} };
    else if (path === "/api/v1/groups/all") data = [{ id: "panel-test", name: "Real collection", currency: "USD" }];
    else if (path === "/api/v1/groups") data = { id: "panel-test", name: "Real collection", currency: "USD" };
    else if (path === "/api/v1/groups/statistics")
      data = { totalItemPrice: 0, totalItems: 7, totalLocations: 1, totalTags: 1 };
    else if (path === "/api/v1/entities" || path === "/api/v1/tags") {
      await sectionsPending;
      if (mode === "failed") {
        await route.fulfill({ status: 500, json: { error: "Test failure" } });
        return;
      }
      if (path === "/api/v1/entities")
        data = { items: mode === "empty" ? [] : url.searchParams.has("isLocation") ? [location] : items, total: 7 };
      else
        data =
          mode === "empty"
            ? []
            : [
                {
                  id: "custom-tag",
                  name: "Fragile / Handle with care",
                  description: "User-defined handling meaning",
                  color: "#123456",
                  icon: "toolbox-outline",
                },
              ];
    }
    await route.fulfill({ json: data });
  });

  await page.goto("/home");
  const recent = page.getByRole("region", { name: "Recently Added", exact: true }).first();
  const locations = page.getByRole("region", { name: "Storage Locations", exact: true });
  const tags = page.getByRole("region", { name: "Tags", exact: true });
  const rows = recent.locator("tbody tr");
  for (const section of [recent, locations, tags]) await expect(section.getByRole("status")).toContainText("Loading");
  finishSections();
  await expect(rows).toHaveCount(5);
  await expect(recent.locator("thead th")).toHaveText(["Name", "Asset ID", "Location", "Quantity"]);
  await expect(rows.first()).toContainText(longName);
  await expect(rows.first().locator("td").nth(1)).toHaveText("000-107");
  await expect(rows.first().locator("td").nth(2)).toHaveText("No location assigned");
  await expect(rows.first().locator("td").nth(3)).toHaveText("0");
  await expect(rows.nth(4)).toContainText("Inventory 4");
  await expect(rows.first().getByRole("link")).toHaveAttribute("href", "/item/record-0");
  await expect(rows.nth(1).getByRole("link", { name: location.name })).toHaveAttribute(
    "href",
    "/location/real-location"
  );
  await expect(locations.getByRole("link", { name: /用户 Storage/ })).toContainText("0 inventory records");
  await expect(locations.getByRole("link", { name: /用户 Storage/ })).toHaveAttribute(
    "href",
    "/location/real-location"
  );
  await expect(recent.getByRole("link", { name: "Search inventory" })).toHaveAttribute("href", "/items");
  await expect(locations.getByRole("link", { name: "View all" })).toHaveAttribute("href", "/locations");
  await expect(tags.getByRole("link", { name: "Manage" })).toHaveAttribute("href", "/tags");
  const chip = tags.getByRole("link", { name: "Fragile / Handle with care" });
  await expect(chip).toHaveAttribute("href", "/tag/custom-tag");
  await expect(chip).toHaveAttribute("title", "User-defined handling meaning");
  await expect(chip).toHaveCSS("background-color", "rgb(18, 52, 86)");
  expect((await locations.boundingBox())!.y).toBe((await tags.boundingBox())!.y);
  const query = requests.find(url => url.pathname === "/api/v1/entities" && !url.searchParams.has("isLocation"));
  expect(query?.searchParams.get("pageSize")).toBe("5");
  expect(query?.searchParams.get("orderBy")).toBe("createdAt");
  await rows.first().getByRole("link").focus();
  await expect(rows.first().getByRole("link")).toBeFocused();
  await page.screenshot({ path: test.info().outputPath("inventory-desktop.png"), fullPage: true });

  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await expect(rows.first().getByRole("link")).toBeVisible();
  await page.screenshot({ path: test.info().outputPath("inventory-narrow.png"), fullPage: true });

  mode = "failed";
  await page.reload();
  await expect(recent.getByRole("alert")).toContainText("Recent inventory could not be loaded");
  await expect(locations.getByRole("alert")).toContainText("Storage locations could not be loaded");
  await expect(tags.getByRole("alert")).toContainText("Tags could not be loaded");
  mode = "empty";
  for (const section of [recent, locations, tags])
    await section.getByRole("button", { name: "Retry", exact: true }).click();
  for (const section of [recent, locations, tags]) await expect(section.getByRole("alert")).toHaveCount(0);
  await expect(rows).toHaveCount(0);
  await expect(locations.getByRole("link", { name: "View all" })).toBeVisible();
  await expect(tags.getByRole("link", { name: "Manage" })).toBeVisible();
  await expect(page.getByText(/Illustrative data|Desktop design concept|Packaging supplies/)).toHaveCount(0);
});
