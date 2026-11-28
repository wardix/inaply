import { describe, expect, test } from "bun:test";
import { createSignageService } from "./signage";

describe("Signage Multi-Zone Schedule", () => {
  test("selects playlist per zone according to property timezone and active hours", () => {
    const svc = createSignageService();

    svc.addSchedule({
      propertyId: "prop-1",
      timezone: "Asia/Jakarta", // UTC+7
      zone: "restaurant",
      startHour: 6,
      endHour: 11,
      playlistId: "breakfast-buffet",
    });
    svc.addSchedule({
      propertyId: "prop-1",
      timezone: "Asia/Jakarta",
      zone: "restaurant",
      startHour: 11,
      endHour: 15,
      playlistId: "lunch-special",
    });
    svc.addSchedule({
      propertyId: "prop-1",
      timezone: "Asia/Jakarta",
      zone: "lobby",
      startHour: 0,
      endHour: 24,
      playlistId: "lobby-welcome",
    });

    // 2026-11-28T01:30:00Z = 08:30 WIB (Asia/Jakarta) -> breakfast-buffet
    const morningRestaurant = svc.resolveActivePlaylist(
      "prop-1",
      "restaurant",
      new Date("2026-11-28T01:30:00Z")
    );
    expect(morningRestaurant).toBe("breakfast-buffet");

    // 2026-11-28T05:00:00Z = 12:00 WIB (Asia/Jakarta) -> lunch-special
    const noonRestaurant = svc.resolveActivePlaylist(
      "prop-1",
      "restaurant",
      new Date("2026-11-28T05:00:00Z")
    );
    expect(noonRestaurant).toBe("lunch-special");

    // Lobby at same time -> lobby-welcome
    const lobby = svc.resolveActivePlaylist(
      "prop-1",
      "lobby",
      new Date("2026-11-28T05:00:00Z")
    );
    expect(lobby).toBe("lobby-welcome");

    // Multi-tenant isolation: prop-2 must NOT see prop-1 schedules
    expect(
      svc.resolveActivePlaylist("prop-2", "lobby", new Date("2026-11-28T05:00:00Z"))
    ).toBeNull();

    // Boundary: endHour (11:00 WIB = 04:00Z) is exclusive -> switches to lunch-special
    expect(
      svc.resolveActivePlaylist("prop-1", "restaurant", new Date("2026-11-28T04:00:00Z"))
    ).toBe("lunch-special");
  });
});
