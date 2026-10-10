import { inject, provide, type InjectionKey } from "vue";

const workspaceSurfaceKey: InjectionKey<string> = Symbol("workspace-surface");

/** Layout-owned scope, retained by Vue injection even when Reka teleports to body. */
export function provideWorkspaceSurface() {
  provide(workspaceSurfaceKey, "owd-workspace");
}

/** Public/login consumers get no scope and keep their selected theme. */
export function useWorkspaceSurface() {
  return inject(workspaceSurfaceKey, "");
}
