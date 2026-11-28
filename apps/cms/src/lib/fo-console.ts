export type GuestSegment = "regular" | "vip" | "corporate" | "group" | string;

export interface FrontOfficeActor {
  userId: string;
  role: string;
  propertyId: string;
}

export interface RoomState {
  roomId: string;
  roomNumber: string;
  propertyId: string;
  status: "vacant" | "occupied";
  guestName?: string;
  segment?: GuestSegment;
  checkoutAt?: string;
  pairedDevices: number;
}

export type GatewaySyncEvent =
  | {
      type: "checkin";
      propertyId: string;
      roomId: string;
      guestName: string;
      segment?: GuestSegment;
      checkoutAt?: string;
    }
  | {
      type: "extend";
      propertyId: string;
      roomId: string;
      checkoutAt: string;
    }
  | {
      type: "checkout";
      propertyId: string;
      roomId: string;
    };

export interface AuditLogEntry {
  actor_id: string;
  room_id: string;
  action: "check_in" | "room_move" | "extend_stay" | "check_out";
  timestamp: string;
}

export class ForbiddenError extends Error {
  readonly status = 403;
  constructor(message = "Forbidden") {
    super(message);
    this.name = "ForbiddenError";
  }
}

export interface FrontOfficeServiceOptions {
  rooms: Map<string, RoomState>;
  onGatewaySync?: (event: GatewaySyncEvent) => void;
  onAuditLog?: (entry: AuditLogEntry) => void;
  now?: () => string;
}

const ALLOWED_FO_ROLES = new Set(["hotel_fo", "hotel_gm", "super_admin"]);

export function createFrontOfficeService(opts: FrontOfficeServiceOptions) {
  const { rooms, onGatewaySync, onAuditLog } = opts;
  const now = opts.now ?? (() => new Date().toISOString());

  function assertAuthorized(actor: FrontOfficeActor, room: RoomState): void {
    if (!ALLOWED_FO_ROLES.has(actor.role)) {
      throw new ForbiddenError(`Role ${actor.role} cannot perform Front Office actions`);
    }
    if (actor.propertyId !== room.propertyId) {
      throw new ForbiddenError("Cross-property access denied");
    }
  }

  function getRoomOrThrow(roomId: string): RoomState {
    const room = rooms.get(roomId);
    if (!room) {
      throw new Error(`Room not found: ${roomId}`);
    }
    return room;
  }

  function recordAudit(
    actor: FrontOfficeActor,
    roomId: string,
    action: AuditLogEntry["action"]
  ): void {
    onAuditLog?.({
      actor_id: actor.userId,
      room_id: roomId,
      action,
      timestamp: now(),
    });
  }

  return {
    listRooms(actor: FrontOfficeActor): RoomState[] {
      const result: RoomState[] = [];
      for (const room of rooms.values()) {
        if (room.propertyId === actor.propertyId) {
          result.push({ ...room });
        }
      }
      return result;
    },

    checkIn(
      actor: FrontOfficeActor,
      input: {
        roomId: string;
        guestName: string;
        segment?: GuestSegment;
        checkoutAt?: string;
      }
    ): RoomState {
      const room = getRoomOrThrow(input.roomId);
      assertAuthorized(actor, room);

      room.status = "occupied";
      room.guestName = input.guestName;
      room.segment = input.segment;
      room.checkoutAt = input.checkoutAt;

      onGatewaySync?.({
        type: "checkin",
        propertyId: room.propertyId,
        roomId: room.roomId,
        guestName: input.guestName,
        segment: input.segment,
        checkoutAt: input.checkoutAt,
      });
      recordAudit(actor, room.roomId, "check_in");
      return { ...room };
    },

    roomMove(
      actor: FrontOfficeActor,
      input: { fromRoomId: string; toRoomId: string }
    ): { fromRoom: RoomState; toRoom: RoomState } {
      const fromRoom = getRoomOrThrow(input.fromRoomId);
      const toRoom = getRoomOrThrow(input.toRoomId);
      assertAuthorized(actor, fromRoom);
      assertAuthorized(actor, toRoom);

      if (fromRoom.status !== "occupied" || !fromRoom.guestName) {
        throw new Error(`Source room ${fromRoom.roomId} is not occupied`);
      }

      const guestName = fromRoom.guestName;
      const segment = fromRoom.segment;
      const checkoutAt = fromRoom.checkoutAt;

      // Vacate source room
      fromRoom.status = "vacant";
      fromRoom.guestName = undefined;
      fromRoom.segment = undefined;
      fromRoom.checkoutAt = undefined;
      fromRoom.pairedDevices = 0;

      // Occupy target room
      toRoom.status = "occupied";
      toRoom.guestName = guestName;
      toRoom.segment = segment;
      toRoom.checkoutAt = checkoutAt;

      onGatewaySync?.({
        type: "checkout",
        propertyId: fromRoom.propertyId,
        roomId: fromRoom.roomId,
      });
      onGatewaySync?.({
        type: "checkin",
        propertyId: toRoom.propertyId,
        roomId: toRoom.roomId,
        guestName,
        segment,
        checkoutAt,
      });
      recordAudit(actor, toRoom.roomId, "room_move");

      return { fromRoom: { ...fromRoom }, toRoom: { ...toRoom } };
    },

    extendStay(
      actor: FrontOfficeActor,
      input: { roomId: string; newCheckoutAt: string }
    ): RoomState {
      const room = getRoomOrThrow(input.roomId);
      assertAuthorized(actor, room);

      if (room.status !== "occupied") {
        throw new Error(`Room ${room.roomId} is not occupied`);
      }

      room.checkoutAt = input.newCheckoutAt;
      onGatewaySync?.({
        type: "extend",
        propertyId: room.propertyId,
        roomId: room.roomId,
        checkoutAt: input.newCheckoutAt,
      });
      recordAudit(actor, room.roomId, "extend_stay");
      return { ...room };
    },

    checkOut(
      actor: FrontOfficeActor,
      input: { roomId: string }
    ): RoomState {
      const room = getRoomOrThrow(input.roomId);
      assertAuthorized(actor, room);

      room.status = "vacant";
      room.guestName = undefined;
      room.segment = undefined;
      room.checkoutAt = undefined;
      room.pairedDevices = 0;

      onGatewaySync?.({
        type: "checkout",
        propertyId: room.propertyId,
        roomId: room.roomId,
      });
      recordAudit(actor, room.roomId, "check_out");
      return { ...room };
    },
  };
}
