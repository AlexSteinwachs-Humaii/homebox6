import { describe, expect, it } from "vitest";
import { createSSRApp, defineComponent, h, Teleport } from "vue";
import { renderToString } from "vue/server-renderer";
import { provideWorkspaceSurface, useWorkspaceSurface } from "./useWorkspaceSurface";

const Surface = defineComponent({
  setup() {
    const scope = useWorkspaceSurface();
    return () => h("section", { class: scope }, "surface");
  },
});

function workspace(child: ReturnType<typeof h>) {
  return defineComponent({
    setup() {
      provideWorkspaceSurface();
      return () => child;
    },
  });
}

describe("workspace surface scope", () => {
  it("does not brand public consumers of shared components", async () => {
    expect(await renderToString(createSSRApp(Surface))).toBe('<section class="">surface</section>');
  });

  it("brands workspace descendants", async () => {
    expect(await renderToString(createSSRApp(workspace(h(Surface))))).toContain('class="owd-workspace"');
  });

  it("retains scope when a surface teleports outside the layout", async () => {
    const context: { teleports?: Record<string, string> } = {};
    await renderToString(createSSRApp(workspace(h(Teleport, { to: "body" }, [h(Surface)]))), context);
    expect(context.teleports?.body).toContain('class="owd-workspace"');
  });

  it("does not leak to a public sibling or a later application", async () => {
    const app = createSSRApp({
      render: () => h("main", [h(workspace(h(Surface))), h(Surface)]),
    });
    const html = await renderToString(app);
    expect(html).toContain('<section class="owd-workspace">surface</section><section class="">surface</section>');
    expect(await renderToString(createSSRApp(Surface))).not.toContain("owd-workspace");
  });
});
