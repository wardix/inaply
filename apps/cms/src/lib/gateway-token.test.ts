import { describe, expect, test } from "bun:test";
import { GatewayTokenService } from "./gateway-token";

describe("Gateway Registration Token", () => {
  test("hashes token before storing and is valid only once before expiry", () => {
    let nowMs = 1_700_000_000_000;
    const svc = new GatewayTokenService(() => nowMs);

    const issued = svc.issueToken("prop-a", 60_000);
    expect(issued.rawToken.length).toBeGreaterThan(16);
    expect(issued.tokenHash).not.toBe(issued.rawToken);

    // Valid once before expiry
    expect(svc.consumeToken(issued.rawToken)).toEqual({ valid: true, propertyId: "prop-a" });
    // Replay fails
    expect(svc.consumeToken(issued.rawToken)).toEqual({ valid: false, reason: "already_used" });

    // Expired token fails
    const issued2 = svc.issueToken("prop-a", 1_000);
    nowMs += 2_000;
    expect(svc.consumeToken(issued2.rawToken)).toEqual({ valid: false, reason: "expired" });
  });
});
