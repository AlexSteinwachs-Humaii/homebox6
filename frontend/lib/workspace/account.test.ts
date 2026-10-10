import { describe, expect, it } from "vitest";
import { accountInitials } from "./account";

describe("workspace account initials", () => {
  it("uses the actual first and last names", () => {
    expect(accountInitials("  Jamie   Taylor Smith  ")).toBe("JS");
  });
  it("handles single and missing names without sample account data", () => {
    expect(accountInitials("Morgan")).toBe("M");
    expect(accountInitials("   ")).toBe("?");
    expect(accountInitials(null)).toBe("?");
    expect(accountInitials()).toBe("?");
  });
  it("retains unicode initials", () => {
    expect(accountInitials("Élodie 王")).toBe("É王");
    expect(accountInitials("𠮷田 太郎")).toBe("𠮷太");
  });
});
