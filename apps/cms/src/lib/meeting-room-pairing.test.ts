import { describe, expect, test } from "bun:test";
import { createMeetingRoomService } from "./meeting-room";

describe("Meeting Room Pairing (Inaply Present)", () => {
  test("supports temporary session code with expires_at and disconnects previous presenter on expiry", () => {
    const svc = createMeetingRoomService();
    const t0 = new Date("2026-11-28T09:00:00Z");
    const expiresAt = new Date("2026-11-28T10:00:00Z");

    const session = svc.startMeetingSession({
      propertyId: "prop-1",
      meetingRoomId: "ballroom-a",
      sessionCode: "CONF-2026",
      expiresAt,
      backdropTitle: "Annual Tech Summit",
    });

    svc.pairPresenter("ballroom-a", "CONF-2026", "presenter-1", t0);
    expect(svc.getActivePresenters("ballroom-a")).toEqual(["presenter-1"]);

    // After expiresAt -> expireStale disconnects presenter-1 automatically
    const expiredCount = svc.expireStale(new Date("2026-11-28T10:00:01Z"));
    expect(expiredCount).toBe(1);
    expect(svc.getActivePresenters("ballroom-a")).toEqual([]);
    expect(() =>
      svc.pairPresenter("ballroom-a", session.sessionCode, "presenter-2", new Date("2026-11-28T10:05:00Z"))
    ).toThrow();
  });
});
