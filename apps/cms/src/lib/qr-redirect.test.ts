import { describe, expect, test } from "bun:test";
import { createAnalyticsService } from "./analytics";

describe("QR Redirect Tracker", () => {
  test("records scan per property and returns 302 only to registered cta_url (prevents open-redirect)", () => {
    const svc = createAnalyticsService({
      registeredAds: new Map([
        [
          "ad-spa-1",
          {
            ad_id: "ad-spa-1",
            property_id: "prop-1",
            cta_url: "https://hotel.example.com/spa-promo",
          },
        ],
      ]),
    });

    // Valid registered ad -> 302 to registered cta_url & records click
    const valid = svc.trackQrRedirect({
      ad_id: "ad-spa-1",
      property_id: "prop-1",
      redirect_to: "https://evil.attacker.com/phish", // must be ignored!
    });
    expect(valid.status).toBe(302);
    expect(valid.location).toBe("https://hotel.example.com/spa-promo");
    expect(svc.getQrClicks("prop-1", "ad-spa-1")).toBe(1);

    // Unregistered ad or cross-property ad -> 404 (no open redirect)
    const unknown = svc.trackQrRedirect({
      ad_id: "ad-unknown",
      property_id: "prop-1",
      redirect_to: "https://evil.attacker.com/phish",
    });
    expect(unknown.status).toBe(404);
    expect(unknown.location).toBeUndefined();
  });
});
