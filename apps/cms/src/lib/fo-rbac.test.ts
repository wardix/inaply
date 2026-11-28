import { describe, expect, test } from "bun:test";
import {
  createFrontOfficeService,
  ForbiddenError,
  type FrontOfficeActor,
  type RoomState,
} from "./fo-console";

describe("Front Office RBAC", () => {
  test("hotel_marketing is forbidden from triggering guest checkout", () => {
    const rooms = new Map<string, RoomState>([
      [
        "room-301",
        {
          roomId: "room-301",
          roomNumber: "301",
          propertyId: "prop-1",
          status: "occupied",
          guestName: "Alice",
          pairedDevices: 1,
        },
      ],
    ]);

    const svc = createFrontOfficeService({ rooms });
    const marketingUser: FrontOfficeActor = {
      userId: "user-mkt-1",
      role: "hotel_marketing",
      propertyId: "prop-1",
    };

    expect(() => svc.checkOut(marketingUser, { roomId: "room-301" })).toThrow(
      ForbiddenError
    );
  });
});
