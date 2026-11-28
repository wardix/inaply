import { describe, expect, test } from "bun:test";
import { OnboardingStore } from "./onboarding";

describe("Property -> Gateway -> Room -> Receiver Onboarding", () => {
  test("validates room_number uniqueness per property and global device_id uniqueness", () => {
    const store = new OnboardingStore();

    const propA = store.createProperty({ partnerId: "partner-1", name: "Hotel A" });
    const propB = store.createProperty({ partnerId: "partner-1", name: "Hotel B" });

    const gwA = store.createGateway({ propertyId: propA.id, name: "GW-A" });
    const room301A = store.createRoom({ propertyId: propA.id, roomNumber: "301" });
    expect(room301A.roomNumber).toBe("301");

    // Same room_number in same property -> rejected
    expect(() => store.createRoom({ propertyId: propA.id, roomNumber: "301" })).toThrow();

    // Same room_number in different property -> allowed
    const room301B = store.createRoom({ propertyId: propB.id, roomNumber: "301" });
    expect(room301B.roomNumber).toBe("301");

    // Receiver with global unique device_id
    const rcv1 = store.createReceiver({ gatewayId: gwA.id, roomId: room301A.id, deviceId: "tv-301-main" });
    expect(rcv1.deviceId).toBe("tv-301-main");

    // Duplicate global device_id -> rejected
    expect(() =>
      store.createReceiver({ gatewayId: gwA.id, roomId: room301B.id, deviceId: "tv-301-main" })
    ).toThrow();
  });
});
