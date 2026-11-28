import { describe, expect, test } from "bun:test";
import { createAnalyticsService } from "./analytics";

describe("Telemetry Privacy", () => {
  test("strips/rejects viewing PII and stores only anonymous aggregates", () => {
    const svc = createAnalyticsService();

    // Payload containing viewing PII fields
    const res = svc.ingestTelemetry({
      property_id: "prop-1",
      protocol: "google_cast",
      session_duration: 1800,
      ad_id: "ad-spa-1",
      timestamp: "2026-11-28T14:00:00Z",
      // PII fields that must NEVER be stored
      video_url: "https://private.example.com/watch?v=secret",
      app_name: "PrivateBrowser",
      media_title: "My Personal Video",
    } as Record<string, unknown>);

    expect(res.accepted).toBe(true);
    const stored = svc.getStoredTelemetry("prop-1");
    expect(stored).toHaveLength(1);
    expect(stored[0]).toEqual({
      property_id: "prop-1",
      protocol: "google_cast",
      session_duration: 1800,
      ad_id: "ad-spa-1",
      timestamp: "2026-11-28T14:00:00Z",
    });
    expect("video_url" in stored[0]).toBe(false);
    expect("app_name" in stored[0]).toBe(false);
    expect("media_title" in stored[0]).toBe(false);
  });
});
