import { describe, expect, test } from "bun:test";
import {
  createFrontOfficeService,
  type AuditLogEntry,
  type FrontOfficeActor,
  type RoomState,
} from "./fo-console";

describe("Front Office Audit Log", () => {
  test("every Front Office action records actor_id, room_id, action, and timestamp", () => {
    const auditLogs: AuditLogEntry[] = [];
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
      onAuditLog: (entry) => auditLogs.push(entry),
      now: () => "2026-11-28T10:00:00.000Z",
    });

    const actor: FrontOfficeActor = {
      userId: "user-fo-1",
      role: "hotel_fo",
      propertyId: "prop-1",
    };

    svc.checkIn(actor, { roomId: "room-301", guestName: "Budi", segment: "corporate" });
    svc.roomMove(actor, { fromRoomId: "room-301", toRoomId: "room-302" });
    svc.extendStay(actor, { roomId: "room-302", newCheckoutAt: "2026-12-02T12:00:00Z" });
    svc.checkOut(actor, { roomId: "room-302" });

    expect(auditLogs).toHaveLength(4);
    expect(auditLogs.map((l) => l.action)).toEqual([
      "check_in",
      "room_move",
      "extend_stay",
      "check_out",
    ]);
    for (const entry of auditLogs) {
      expect(entry.actor_id).toBe("user-fo-1");
      expect(entry.room_id).toBeTruthy();
      expect(entry.timestamp).toBe("2026-11-28T10:00:00.000Z");
    }
  });
});
