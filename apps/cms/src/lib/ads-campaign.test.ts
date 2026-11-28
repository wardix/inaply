import { describe, expect, test } from "bun:test";
import { createAdCampaign, filterActiveAdsForSegment, type AdCampaign } from "./ads-campaign";

describe("TV Ads Campaign", () => {
  test("rejects ends_at <= starts_at, validates media MIME type, and filters by target_segments", () => {
    const start = new Date("2026-11-28T00:00:00Z");
    const end = new Date("2026-11-29T00:00:00Z");

    expect(() =>
      createAdCampaign({
        propertyId: "prop-a",
        title: "Invalid Schedule",
        mediaUrl: "https://cdn.inaply.local/ad.mp4",
        mimeType: "video/mp4",
        targetSegments: ["vip"],
        startsAt: start,
        endsAt: start,
      })
    ).toThrow();

    expect(() =>
      createAdCampaign({
        propertyId: "prop-a",
        title: "Bad MIME",
        mediaUrl: "https://cdn.inaply.local/script.exe",
        mimeType: "application/x-msdownload",
        targetSegments: ["leisure"],
        startsAt: start,
        endsAt: end,
      })
    ).toThrow();

    const vipAd = createAdCampaign({
      propertyId: "prop-a",
      title: "VIP Spa",
      mediaUrl: "https://cdn.inaply.local/spa.mp4",
      mimeType: "video/mp4",
      targetSegments: ["vip"],
      startsAt: start,
      endsAt: end,
    });

    const allAd = createAdCampaign({
      propertyId: "prop-a",
      title: "Breakfast",
      mediaUrl: "https://cdn.inaply.local/breakfast.jpg",
      mimeType: "image/jpeg",
      targetSegments: [],
      startsAt: start,
      endsAt: end,
    });

    const campaigns: AdCampaign[] = [vipAd, allAd];
    const now = new Date("2026-11-28T12:00:00Z");

    expect(filterActiveAdsForSegment(campaigns, "corporate", now).map((a) => a.title)).toEqual([
      "Breakfast",
    ]);
    expect(filterActiveAdsForSegment(campaigns, "vip", now).map((a) => a.title)).toEqual([
      "VIP Spa",
      "Breakfast",
    ]);
  });
});
