import { describe, expect, test } from "bun:test";
import {
  createFrontOfficeService,
  type FrontOfficeActor,
  type GatewaySyncEvent,
  type RoomState,
} from "./fo-console";

describe("Front Office Console API", () => {
  const actor: FrontOfficeActor = {
    userId: "user-fo-1",
    role: "hotel_fo",
    propertyId: "prop-1",
  };

  test("check-in, room move, extend stay, and check-out emit gateway sync events", () => {
    const emitted: GatewaySyncEvent[] = [];
    const rooms = new Map<string, RoomState>([
      [
        "room-301",
        {
          roomId: "room-301",
          roomNumber: "301",
          propertyId: "prop-1",
          status: "vacant",
          pairedDevices: 0,
        },
      ],
      [
        "room-302",
        {
          roomId: "room-302",
          roomNumber: "302",
          propertyId: "prop-1",
          status: "vacant",
          pairedDevices: 0,
        },
      ],
    ]);

    const svc = createFrontOfficeService({
      rooms,
      onGatewaySync: (ev) => emitted.push(ev),
    });

    // 1. Check-in
    const checkedIn = svc.checkIn(actor, {
      roomId: "room-301",
      guestName: "Alice",
      segment: "vip",
      checkoutAt: "2026-11-30T12:00:00Z",
    });
    expect(checkedIn.status).toBe("occupied");
    expect(checkedIn.guestName).toBe("Alice");
    expect(emitted[0]).toMatchObject({
      type: "checkin",
      propertyId: "prop-1",
      roomId: "room-301",
      guestName: "Alice",
      segment: "vip",
    });

    // 2. Room Move (301 -> 302): revokes old room & checks in target room
    const moved = svc.roomMove(actor, {
      fromRoomId: "room-301",
      toRoomId: "room-302",
    });
    expect(moved.fromRoom.status).toBe("vacant");
    expect(moved.toRoom.status).toBe("occupied");
    expect(moved.toRoom.guestName).toBe("Alice");
    expect(emitted[1]).toMatchObject({
      type: "checkout",
      propertyId: "prop-1",
      roomId: "room-301",
    });
    expect(emitted[2]).toMatchObject({
      type: "checkin",
      propertyId: "prop-1",
      roomId: "room-302",
      guestName: "Alice",
    });

    // 3. Extend Stay
    const extended = svc.extendStay(actor, {
      roomId: "room-302",
      newCheckoutAt: "2026-12-01T12:00:00Z",
    });
    expect(extended.checkoutAt).toBe("2026-12-01T12:00:00Z");
    expect(emitted[3]).toMatchObject({
      type: "extend",
      propertyId: "prop-1",
      roomId: "room-302",
      checkoutAt: "2026-12-01T12:00:00Z",
    });

    // 4. Check-out & Wipe
    const checkedOut = svc.checkOut(actor, { roomId: "room-302" });
    expect(checkedOut.status).toBe("vacant");
    expect(checkedOut.guestName).toBeUndefined();
    expect(emitted[4]).toMatchObject({
      type: "checkout",
      propertyId: "prop-1",
      roomId: "room-302",
    });
  });
});
