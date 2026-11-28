import { describe, expect, test } from "bun:test";
import { authenticateGatewaySync } from "./sync-auth";

describe("Cloud-to-Edge Sync Auth", () => {
  test("rejects missing, invalid, or cross-property gateway token with 401", () => {
    const registry = new Map<string, { propertyId: string; token: string }>([
      ["gw-prop-a", { propertyId: "prop-a", token: "secret-a" }],
    ]);

    expect(authenticateGatewaySync("gw-prop-a", "prop-a", "", registry).status).toBe(401);
    expect(authenticateGatewaySync("gw-prop-a", "prop-a", "wrong", registry).status).toBe(401);
    expect(authenticateGatewaySync("gw-prop-a", "prop-b", "secret-a", registry).status).toBe(401);
    expect(authenticateGatewaySync("gw-prop-a", "prop-a", "secret-a", registry).status).toBe(200);
  });
});
