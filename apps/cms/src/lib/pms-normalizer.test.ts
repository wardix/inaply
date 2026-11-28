import { describe, expect, test } from "bun:test";
import {
  CloudbedsAdapter,
  LittleHotelierAdapter,
  OperaAdapter,
  RealtaAdapter,
  VHPAdapter,
  type PMSAdapter,
} from "./pms-adapter";

describe("PMS Normalizer", () => {
  test("maps VHP, Realta, Cloudbeds, Little Hotelier, and Oracle Opera payloads into uniform PMSEvent", () => {
    const cases: Array<{ adapter: PMSAdapter; payload: Record<string, unknown>; expectedType: string }> = [
      {
        adapter: new VHPAdapter(),
        payload: {
          event: "CHECK_IN",
          resv_no: "VHP-101",
          room_no: "301",
          guest_name: "Budi Santoso",
          vip_code: "vip",
          dep_date: "2026-11-30T12:00:00Z",
        },
        expectedType: "CheckIn",
      },
      {
        adapter: new RealtaAdapter(),
        payload: {
          action: "ROOM_MOVE",
          reservation_id: "RLT-202",
          old_room: "301",
          new_room: "302",
          guest: "Siti Aminah",
        },
        expectedType: "RoomMove",
      },
      {
        adapter: new CloudbedsAdapter(),
        payload: {
          event_type: "reservation/dates_changed",
          reservationID: "CB-303",
          roomName: "501",
          endDate: "2026-12-02T12:00:00Z",
        },
        expectedType: "UpdateStay",
      },
      {
        adapter: new LittleHotelierAdapter(),
        payload: {
          type: "guest.checked_out",
          booking_ref: "LH-404",
          room_number: "301",
        },
        expectedType: "CheckOut",
      },
      {
        adapter: new OperaAdapter(),
        payload: {
          eventType: "CHECKIN",
          reservationId: "OPR-505",
          roomId: "502",
          guestName: "John Doe",
          guestSegment: "corporate",
          checkoutTime: "2026-12-01T11:00:00Z",
        },
        expectedType: "CheckIn",
      },
    ];

    for (const c of cases) {
      const ev = c.adapter.normalize(c.payload);
      expect(ev.type).toBe(c.expectedType);
      expect(ev.pms_resv_id).toBeTruthy();
      expect(ev.room_number).toBeTruthy();
    }
  });
});
