import { describe, expect, test } from "bun:test";
import { sendGuestMessage } from "./guest-messages";

describe("Guest Messages", () => {
  test("marks room_id=null as property broadcast and rejects cross-property room_id with 403", () => {
    const roomPropertyMap = new Map<string, string>([
      ["room-301", "prop-a"],
      ["room-302", "prop-b"],
    ]);

    const broadcast = sendGuestMessage(
      {
        propertyId: "prop-a",
        roomId: null,
        title: "Pool Maintenance",
        body: "Pool closes at 10 PM",
        priority: "normal",
        showAsOverlay: true,
      },
      roomPropertyMap
    );
    expect(broadcast.status).toBe(200);
    if (broadcast.status === 200) {
      expect(broadcast.message.isBroadcast).toBe(true);
    }

    const crossProperty = sendGuestMessage(
      {
        propertyId: "prop-a",
        roomId: "room-302",
        title: "Cross Tenant",
        body: "Should fail",
        priority: "urgent",
        showAsOverlay: true,
      },
      roomPropertyMap
    );
    expect(crossProperty.status).toBe(403);
  });
});
