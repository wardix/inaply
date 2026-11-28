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

  test("late/out-of-order RoomMove does not checkout a new guest in from_room_number", () => {
    const syncCalls: Array<{ action: string; room_number: string; guest_name?: string }> = [];
    const proc = createPMSProcessor({
      onSync: (cmd) => syncCalls.push(cmd),
    });

    // New guest RESV-NEW is already checked into 301
    proc.process("prop-1", {
      type: "CheckIn",
      pms_resv_id: "RESV-NEW",
      room_number: "301",
      guest_name: "NewGuest",
    });

    // Late RoomMove for old RESV-OLD from 301 -> 302 must NOT touch 301
    proc.process("prop-1", {
      type: "RoomMove",
      pms_resv_id: "RESV-OLD",
      from_room_number: "301",
      room_number: "302",
      guest_name: "OldGuest",
    });

    expect(proc.getActiveReservation("prop-1", "301")?.pms_resv_id).toBe("RESV-NEW");
    expect(syncCalls).toEqual([
      { action: "checkin", room_number: "301", guest_name: "NewGuest" },
      { action: "checkin", room_number: "302", guest_name: "OldGuest" },
    ]);
  });
});
