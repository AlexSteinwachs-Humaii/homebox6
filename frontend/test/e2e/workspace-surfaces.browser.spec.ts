import { test, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import postcss from "postcss";
import tailwindcss from "tailwindcss";
import loadConfig from "tailwindcss/loadConfig.js";
import { fileURLToPath } from "node:url";

const tailwindConfig = loadConfig(fileURLToPath(new URL("../../tailwind.config.js", import.meta.url)));

// A surface fixture, not a shell/navigation test: no backend or sample inventory.
const fixture = `
  <div class='owd-workspace'>
    <aside data-sidebar='sidebar' class='bg-sidebar text-sidebar-foreground'>
      <a id='active' href='#' class='bg-accent text-accent-foreground'>Home</a>
    </aside>
    <main class='bg-background-accent'>
      <section id='panel' class='workspace-panel'>
        <p class='workspace-muted'>Inventory workspace</p>
        <button id='action' class='bg-primary text-primary-foreground'>Create</button>
        <a class='workspace-accent-link' href='#'>View all</a>
      </section>
    </main>
  </div>
  <div id='portal' class='owd-workspace bg-popover text-popover-foreground'>Portalled menu</div>
  <div id='dialog' class='owd-workspace bg-background text-foreground'>Portalled dialog</div>
  <div id='public' class='bg-background text-foreground'>Public page</div>
`;
let css: string;
test.beforeAll(async () => {
  const source = await readFile(new URL("../../assets/css/main.css", import.meta.url), "utf8");
  const result = await postcss([
    tailwindcss({ ...tailwindConfig, content: [{ raw: fixture, extension: "html" }] }),
  ]).process(source, { from: undefined });
  css = result.css;
});

for (const theme of ["homebox", "theme-black", "theme-aqua"]) {
  test(`workspace and external surfaces stay readable with ${theme}`, async ({ page }) => {
    await page.setContent(`<html class='${theme}'><head><style>${css}</style></head><body>${fixture}</body></html>`);
    await expect(page.locator("#panel")).toHaveCSS("background-color", "rgb(255, 255, 255)");
    await expect(page.locator("#panel")).toHaveCSS("color", "rgb(36, 38, 41)");
    await expect(page.locator("aside")).toHaveCSS("background-color", "rgb(36, 38, 41)");
    for (const id of ["portal", "dialog"]) {
      await expect(page.locator(`#${id}`)).toHaveCSS("background-color", "rgb(255, 255, 255)");
      await expect(page.locator(`#${id}`)).toHaveCSS("color", "rgb(36, 38, 41)");
    }
    await expect(page.locator("#action")).toHaveCSS("background-color", "rgb(248, 90, 18)");
    await page.locator("#action").focus();
    await expect(page.locator("#action")).toHaveCSS("outline-style", "solid");
    await expect(page.locator("#action")).toHaveCSS("outline-width", "2px");
    await expect(page.locator("html")).toHaveAttribute("class", theme);
    if (theme === "theme-black") {
      await expect(page.locator("#public")).toHaveCSS("background-color", "rgb(0, 0, 0)");
    }
    // Removing the workspace scope restores the original theme immediately.
    await page.locator("#portal").evaluate(el => el.classList.remove("owd-workspace"));
    const publicBackground = await page.locator("#public").evaluate(el => getComputedStyle(el).backgroundColor);
    await expect(page.locator("#portal")).toHaveCSS("background-color", publicBackground);
  });
}
