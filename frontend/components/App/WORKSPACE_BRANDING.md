# One World Direct workspace foundation

## Scope and reuse

`layouts/default.vue` owns the `owd-workspace` scope and calls `provideWorkspaceSurface()`.
No theme storage, preference sync, backend contracts or public layouts are changed.
The selected theme continues to apply outside the authenticated layout.

Home can use the existing HSL-backed Tailwind colors (`bg-card`, `text-foreground`,
`border-border`, etc.) and these classes inside its existing layout slot:

- `workspace-panel`: opaque white panel, border, radius, spacing and subtle shadow.
- `workspace-muted`: secondary text with legible contrast on light surfaces.
- `workspace-border`: neutral border color (combine with a border-width class).
- `workspace-accent-link`: dark orange link with hover underline.
- `--workspace-space` and `--workspace-panel-shadow`: reusable surface primitives.

Keyboard focus uses a dark orange outline on light surfaces and a lighter orange
outline within the charcoal sidebar. Orange buttons use charcoal text, not white.
The header height variables and the existing responsive sidebar primitives remain
unchanged. Navigation arrangement/account presentation are subsequent story work.

## Portals

Shared menu, tooltip, select, popover, dialog, sheet, alert and drawer content opts
in with `useWorkspaceSurface()`. Vue injection follows component ownership across
teleports, so these surfaces retain workspace tokens even when mounted under body.
Public consumers receive an empty class. The direct Tag selector combobox portal
uses the same hook. A new authenticated portal should also opt in at its content
root; do not set workspace variables on html/body or change the user's preference.

## Outstanding original artwork (acceptance criterion not complete)

The repository's image inventory contains HomeBox/public images, not the supplied
original One World Direct circular mark. The referenced design explicitly says its
vector is a reconstruction and must not be used. No such reconstruction or alternate
web logo has been added.

`WorkspaceBrand.vue` accepts `markSrc` for the original artwork. Currently the layout
omits it and shows a text-only OWD fallback; **this is not the approved circular
mark and is not completion of the artwork criterion**. Once the original is supplied,
retain its original SVG/raster format under `assets/images`, import it into the
layout, and pass `:mark-src`. Confirm provenance, actual resolution and legibility
at the 32px icon size and in expanded/collapsed navigation before sign-off.

## Verification

- `pnpm run build`: passed.
- `pnpm run lint:ci`: passed, with the baseline's one allowed formatting warning
  in `locales/zh-TW.test.ts`.
- Targeted Vitest `composables/useWorkspaceSurface.test.ts`: 4 passed (default
  public scope, workspace inheritance, teleport inheritance and sibling isolation).
- Chromium `test/e2e/workspace-surfaces.browser.spec.ts`: 3 passed. These compile
  the real stylesheet and check light panels/portals, charcoal navigation, orange
  actions and focus with HomeBox, black and aqua themes. They are surface fixtures,
  not an authenticated /home or login walkthrough.
- `pnpm run typecheck`: failed with 196 errors; unchanged HEAD also produced 196
  errors. Examples: `components/Entity/CreateModal.vue(638,19): error TS18048:
  'form.location' is possibly 'undefined'`; `components/Item/View/ItemChangeDetails.vue(64,28):
  error TS2339: Property 'location' does not exist on type 'EntitySummary'`.
  No diagnostics name the new brand, scope hook or surface browser test.

Original-artwork verification and live authenticated expanded/collapsed comparison
remain outstanding. Full navigation/responsive regression belongs to later stories.
