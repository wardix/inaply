import { describe, expect, test } from "bun:test";
import { createPMSProcessor } from "./pms-adapter";

describe("PMS Room Move", () => {
  test("room move revokes TV access in old room and activates welcome greeting in new room", () => {
    const syncCalls: Array<{ action: string; room_number: string; guest_name?: string }> = [];
    const proc = createPMSProcessor({
      onSync: (cmd) => syncCalls.push(cmd),
    });

    proc.process("prop-1", {
      type: "CheckIn",
      pms_resv_id: "RESV-300",
      room_number: "301",
      guest_name: "Charlie",
      segment: "vip",
    });

    proc.process("prop-1", {
      type: "RoomMove",
      pms_resv_id: "RESV-300",
      from_room_number: "301",
      room_number: "302",
      guest_name: "Charlie",
      segment: "vip",
    });

    expect(proc.getActiveReservation("prop-1", "301")).toBeUndefined();
    expect(proc.getActiveReservation("prop-1", "302")?.guest_name).toBe("Charlie");
    expect(syncCalls).toEqual([
      { action: "checkin", room_number: "301", guest_name: "Charlie" },
      { action: "checkout", room_number: "301" },
      { action: "checkin", room_number: "302", guest_name: "Charlie" },
    ]);
  });
});
