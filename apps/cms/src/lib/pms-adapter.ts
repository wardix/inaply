export type PMSEventType = "CheckIn" | "RoomMove" | "UpdateStay" | "CheckOut";

export interface PMSEvent {
  type: PMSEventType;
  pms_resv_id: string;
  room_number: string;
  from_room_number?: string;
  guest_name?: string;
  segment?: string;
  checkout_at?: string;
}

export interface PMSAdapter {
  readonly vendor: string;
  normalize(payload: Record<string, unknown>): PMSEvent;
}

function str(val: unknown): string {
  return typeof val === "string" ? val : "";
}

function mapEventType(raw: string, mapping: Record<string, PMSEventType>): PMSEventType {
  const mapped = mapping[raw];
  if (!mapped) {
    throw new Error(`Unsupported PMS event type: ${raw}`);
  }
  return mapped;
}

export class VHPAdapter implements PMSAdapter {
  readonly vendor = "vhp";
  normalize(payload: Record<string, unknown>): PMSEvent {
    const type = mapEventType(str(payload.event), {
      CHECK_IN: "CheckIn",
      ROOM_MOVE: "RoomMove",
      UPDATE_STAY: "UpdateStay",
      CHECK_OUT: "CheckOut",
    });
    return {
      type,
      pms_resv_id: str(payload.resv_no),
      room_number: str(payload.room_no),
      from_room_number: str(payload.old_room_no) || undefined,
      guest_name: str(payload.guest_name) || undefined,
      segment: str(payload.vip_code) || undefined,
      checkout_at: str(payload.dep_date) || undefined,
    };
  }
}

export class RealtaAdapter implements PMSAdapter {
  readonly vendor = "realta";
  normalize(payload: Record<string, unknown>): PMSEvent {
    const type = mapEventType(str(payload.action), {
      CHECK_IN: "CheckIn",
      ROOM_MOVE: "RoomMove",
      EXTEND_STAY: "UpdateStay",
      CHECK_OUT: "CheckOut",
    });
    return {
      type,
      pms_resv_id: str(payload.reservation_id),
      room_number: str(payload.new_room || payload.room),
      from_room_number: str(payload.old_room) || undefined,
      guest_name: str(payload.guest) || undefined,
      segment: str(payload.segment) || undefined,
      checkout_at: str(payload.checkout_date) || undefined,
    };
  }
}

export class CloudbedsAdapter implements PMSAdapter {
  readonly vendor = "cloudbeds";
  normalize(payload: Record<string, unknown>): PMSEvent {
    const type = mapEventType(str(payload.event_type), {
      "reservation/checked_in": "CheckIn",
      "reservation/room_assigned": "RoomMove",
      "reservation/dates_changed": "UpdateStay",
      "reservation/checked_out": "CheckOut",
    });
    return {
      type,
      pms_resv_id: str(payload.reservationID),
      room_number: str(payload.roomName),
      from_room_number: str(payload.previousRoomName) || undefined,
      guest_name: str(payload.guestName) || undefined,
      checkout_at: str(payload.endDate) || undefined,
    };
  }
}

export class LittleHotelierAdapter implements PMSAdapter {
  readonly vendor = "little_hotelier";
  normalize(payload: Record<string, unknown>): PMSEvent {
    const type = mapEventType(str(payload.type), {
      "guest.checked_in": "CheckIn",
      "guest.room_moved": "RoomMove",
      "guest.stay_updated": "UpdateStay",
      "guest.checked_out": "CheckOut",
    });
    return {
      type,
      pms_resv_id: str(payload.booking_ref),
      room_number: str(payload.room_number),
      from_room_number: str(payload.from_room_number) || undefined,
      guest_name: str(payload.guest_name) || undefined,
      checkout_at: str(payload.checkout_at) || undefined,
    };
  }
}

export class OperaAdapter implements PMSAdapter {
  readonly vendor = "opera";
  normalize(payload: Record<string, unknown>): PMSEvent {
    const type = mapEventType(str(payload.eventType), {
      CHECKIN: "CheckIn",
      ROOM_MOVE: "RoomMove",
      UPDATE_RESERVATION: "UpdateStay",
      CHECKOUT: "CheckOut",
    });
    return {
      type,
      pms_resv_id: str(payload.reservationId),
      room_number: str(payload.roomId),
      from_room_number: str(payload.previousRoomId) || undefined,
      guest_name: str(payload.guestName) || undefined,
      segment: str(payload.guestSegment) || undefined,
      checkout_at: str(payload.checkoutTime) || undefined,
    };
  }
}

export interface PMSSyncCommand {
  action: "checkin" | "extend" | "checkout";
  room_number: string;
  guest_name?: string;
}

export interface ActiveReservation {
  pms_resv_id: string;
  room_number: string;
  guest_name?: string;
  segment?: string;
  checkout_at?: string;
}

export function createPMSProcessor(opts: {
  onSync?: (cmd: PMSSyncCommand) => void;
} = {}) {
  const activeByRoom = new Map<string, ActiveReservation>();
  const key = (propertyId: string, roomNumber: string) => `${propertyId}:${roomNumber}`;

  return {
    getActiveReservation(
      propertyId: string,
      roomNumber: string
    ): ActiveReservation | undefined {
      return activeByRoom.get(key(propertyId, roomNumber));
    },

    process(propertyId: string, ev: PMSEvent): { ignored: boolean } {
      if (ev.type === "CheckIn") {
        activeByRoom.set(key(propertyId, ev.room_number), {
          pms_resv_id: ev.pms_resv_id,
          room_number: ev.room_number,
          guest_name: ev.guest_name,
          segment: ev.segment,
          checkout_at: ev.checkout_at,
        });
        opts.onSync?.({
          action: "checkin",
          room_number: ev.room_number,
          guest_name: ev.guest_name,
        });
        return { ignored: false };
      }

      if (ev.type === "RoomMove") {
        const fromRoom = ev.from_room_number;
        let guestName = ev.guest_name;
        let segment = ev.segment;
        let checkoutAt = ev.checkout_at;

        if (fromRoom) {
          const prev = activeByRoom.get(key(propertyId, fromRoom));
          if (prev) {
            guestName = guestName ?? prev.guest_name;
            segment = segment ?? prev.segment;
            checkoutAt = checkoutAt ?? prev.checkout_at;
            activeByRoom.delete(key(propertyId, fromRoom));
          }
          opts.onSync?.({
            action: "checkout",
            room_number: fromRoom,
          });
        }

        activeByRoom.set(key(propertyId, ev.room_number), {
          pms_resv_id: ev.pms_resv_id,
          room_number: ev.room_number,
          guest_name: guestName,
          segment,
          checkout_at: checkoutAt,
        });
        opts.onSync?.({
          action: "checkin",
          room_number: ev.room_number,
          guest_name: guestName,
        });
        return { ignored: false };
      }

      if (ev.type === "UpdateStay") {
        const current = activeByRoom.get(key(propertyId, ev.room_number));
        if (!current || current.pms_resv_id !== ev.pms_resv_id) {
          return { ignored: true };
        }
        current.checkout_at = ev.checkout_at ?? current.checkout_at;
        opts.onSync?.({
          action: "extend",
          room_number: ev.room_number,
        });
        return { ignored: false };
      }

      // CheckOut: idempotent & guarded by pms_resv_id
      const current = activeByRoom.get(key(propertyId, ev.room_number));
      if (!current || current.pms_resv_id !== ev.pms_resv_id) {
        return { ignored: true };
      }
      activeByRoom.delete(key(propertyId, ev.room_number));
      opts.onSync?.({
        action: "checkout",
        room_number: ev.room_number,
      });
      return { ignored: false };
    },
  };
}
