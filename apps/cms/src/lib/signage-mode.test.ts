import { describe, expect, test } from "bun:test";
import { createSignageService, SignagePairingForbiddenError } from "./signage";

describe("Signage Mode", () => {
  test("rejects guest pairing on signage devices and supports landscape/portrait orientation", () => {
    const svc = createSignageService();

    const lobbyDisplay = svc.registerDisplay({
      deviceId: "dev-lobby-1",
      propertyId: "prop-1",
      mode: "signage",
      zone: "lobby",
      orientation: "landscape",
    });
    const liftDisplay = svc.registerDisplay({
      deviceId: "dev-lift-1",
      propertyId: "prop-1",
      mode: "signage",
      zone: "lift",
      orientation: "portrait",
    });

    expect(lobbyDisplay.orientation).toBe("landscape");
    expect(liftDisplay.orientation).toBe("portrait");

    expect(() => svc.pairGuest("dev-lobby-1", "guest-1")).toThrow(
      SignagePairingForbiddenError
    );
  });
});
