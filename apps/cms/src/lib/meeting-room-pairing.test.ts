import { describe, expect, test } from "bun:test";
import { createMeetingRoomService } from "./meeting-room";

describe("Meeting Room Pairing (Inaply Present)", () => {
  test("supports temporary session code with expires_at and disconnects previous presenter on expiry", () => {
    const disconnected: string[] = [];
    const svc = createMeetingRoomService({
      onDisconnect: (presenterId) => disconnected.push(presenterId),
    });
    const t0 = new Date("2026-11-28T09:00:00Z");
    const expiresAt = new Date("2026-11-28T10:00:00Z");

    const session = svc.startMeetingSession({
      propertyId: "prop-1",
      meetingRoomId: "ballroom-a",
      sessionCode: "CONF-2026",
      expiresAt,
      backdropTitle: "Annual Tech Summit",
    });

    expect(svc.getSessionBackdrop("prop-1", "ballroom-a", t0)).toBe("Annual Tech Summit");
    svc.updateBackdrop("prop-1", "ballroom-a", "Keynote Break");
    expect(svc.getSessionBackdrop("prop-1", "ballroom-a", t0)).toBe("Keynote Break");

    svc.pairPresenter("prop-1", "ballroom-a", "CONF-2026", "presenter-1", t0);
    expect(svc.getActivePresenters("prop-1", "ballroom-a", t0)).toEqual(["presenter-1"]);

    // Multi-tenant isolation: prop-2's ballroom-a must NOT collide with prop-1's ballroom-a
    svc.startMeetingSession({
      propertyId: "prop-2",
      meetingRoomId: "ballroom-a",
      sessionCode: "WEDDING-99",
      expiresAt,
      backdropTitle: "Wedding Reception",
    });
    expect(svc.getActivePresenters("prop-1", "ballroom-a", t0)).toEqual(["presenter-1"]);
    expect(svc.getSessionBackdrop("prop-2", "ballroom-a", t0)).toBe("Wedding Reception");

    // After expiresAt -> expireStale disconnects presenter-1 automatically
    const expiredCount = svc.expireStale(new Date("2026-11-28T10:00:01Z"));
    expect(expiredCount).toBe(2);
    expect(disconnected).toEqual(["presenter-1"]);
    expect(svc.getActivePresenters("prop-1", "ballroom-a", new Date("2026-11-28T10:00:01Z"))).toEqual([]);
    expect(() =>
      svc.pairPresenter("prop-1", "ballroom-a", session.sessionCode, "presenter-2", new Date("2026-11-28T10:05:00Z"))
    ).toThrow();
  });
});
