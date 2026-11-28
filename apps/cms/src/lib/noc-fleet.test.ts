import { describe, expect, test } from "bun:test";
import { computeFleetStatus, type FleetDeviceHeartbeat } from "./noc-fleet";

describe("NOC Fleet Monitoring", () => {
  test("marks receiver offline when heartbeat exceeds threshold", () => {
    const nowMs = 1_700_000_100_000;
    const devices: FleetDeviceHeartbeat[] = [
      {
        deviceId: "tv-301-main",
        propertyId: "prop-a",
        appVersion: "1.2.0",
        hdmiCec: true,
        lastSeenMs: nowMs - 30_000, // 30s ago -> online
      },
      {
        deviceId: "tv-302-main",
        propertyId: "prop-a",
        appVersion: "1.2.0",
        hdmiCec: false,
        lastSeenMs: nowMs - 180_000, // 3m ago (> 90s threshold) -> offline
      },
    ];

    const report = computeFleetStatus(devices, nowMs, 90_000);
    expect(report.find((d) => d.deviceId === "tv-301-main")?.status).toBe("online");
    expect(report.find((d) => d.deviceId === "tv-302-main")?.status).toBe("offline");
  });
});
