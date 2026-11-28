import { describe, expect, test } from "bun:test";
import { createAnalyticsService } from "./analytics";

describe("Analytics Report", () => {
  test("daily aggregation of casting adoption & TV Ads impressions matches raw events and is tenant-isolated", () => {
    const svc = createAnalyticsService();

    // Property 1 events on 2026-11-28
    svc.ingestTelemetry({
      property_id: "prop-1",
      protocol: "google_cast",
      session_duration: 600,
      timestamp: "2026-11-28T10:15:00Z",
    });
    svc.ingestTelemetry({
      property_id: "prop-1",
      protocol: "mirroring",
      session_duration: 300,
      ad_id: "ad-1",
      timestamp: "2026-11-28T10:45:00Z",
    });
    svc.ingestTelemetry({
      property_id: "prop-1",
      protocol: "google_cast",
      session_duration: 900,
      ad_id: "ad-1",
      timestamp: "2026-11-28T20:00:00Z",
    });

    // Property 2 event (must NOT leak into Property 1 report)
    svc.ingestTelemetry({
      property_id: "prop-2",
      protocol: "google_cast",
      session_duration: 1200,
      ad_id: "ad-2",
      timestamp: "2026-11-28T10:00:00Z",
    });

    const report1 = svc.getDailyReport("prop-1", "2026-11-28");
    expect(report1.property_id).toBe("prop-1");
    expect(report1.date).toBe("2026-11-28");
    expect(report1.total_sessions).toBe(3);
    expect(report1.protocol_counts).toEqual({
      google_cast: 2,
      mirroring: 1,
    });
    expect(report1.ad_impressions).toBe(2);
    expect(report1.peak_hour_utc).toBe(10);

    const report2 = svc.getDailyReport("prop-2", "2026-11-28");
    expect(report2.total_sessions).toBe(1);
    expect(report2.ad_impressions).toBe(1);
  });
});
