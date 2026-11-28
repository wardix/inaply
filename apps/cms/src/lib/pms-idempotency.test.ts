import { describe, expect, test } from "bun:test";
import { createPMSProcessor, type PMSEvent } from "./pms-adapter";

describe("PMS Idempotency", () => {
  test("duplicate CheckOut with same pms_resv_id is idempotent and does not reset a newly checked-in guest", () => {
    const syncCalls: Array<{ action: string; room_number: string; guest_name?: string }> = [];
    const proc = createPMSProcessor({
      onSync: (cmd) => syncCalls.push(cmd),
    });

    const guest1In: PMSEvent = {
      type: "CheckIn",
      pms_resv_id: "RESV-100",
      room_number: "301",
      guest_name: "Alice",
    };
    const guest1Out: PMSEvent = {
      type: "CheckOut",
      pms_resv_id: "RESV-100",
      room_number: "301",
    };
    const guest2In: PMSEvent = {
      type: "CheckIn",
      pms_resv_id: "RESV-200",
      room_number: "301",
      guest_name: "Bob",
    };

    proc.process("prop-1", guest1In);
    proc.process("prop-1", guest1Out);

    // New guest checks into room 301
    proc.process("prop-1", guest2In);

    // Duplicate CheckOut for RESV-100 arrives late -> must be ignored!
    const dupResult = proc.process("prop-1", guest1Out);
    expect(dupResult.ignored).toBe(true);
    expect(proc.getActiveReservation("prop-1", "301")?.pms_resv_id).toBe("RESV-200");
    expect(syncCalls).toEqual([
      { action: "checkin", room_number: "301", guest_name: "Alice" },
      { action: "checkout", room_number: "301" },
      { action: "checkin", room_number: "301", guest_name: "Bob" },
    ]);
  });
});
