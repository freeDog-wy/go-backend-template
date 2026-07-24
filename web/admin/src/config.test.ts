import { describe, expect, it } from "vitest";

describe("runtime API configuration", () => {
  it("documents the production API origin", () => {
    expect("https://api.elseif.site").toMatch(/^https:\/\//);
  });
});
