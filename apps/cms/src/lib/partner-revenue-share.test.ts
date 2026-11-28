import { describe, expect, test } from "bun:test";
import { createBillingEngine } from "./billing";

describe("Partner Revenue Share", () => {
  test("calculates deterministic integer wholesale pricing across partner properties without float rounding errors", () => {
    const engine = createBillingEngine();

    // Partner ISP-1 has 20% reseller discount (2000 bps)
    // Property A: 10 rooms * 30.000 = 300.000 -> 20% off = 240.000
    // Property B: 5 rooms * 30.000 = 150.000 -> 20% off = 120.000
    // Consolidated total = 360.000
    const invoice = engine.generatePartnerInvoice({
      partnerId: "partner-isp-1",
      period: "2026-11",
      discountBps: 2000,
      properties: [
        {
          propertyId: "prop-a",
          activeRooms: 10,
          unitPriceMinor: 30_000,
          cycle: "monthly",
          activeDays: 30,
          totalDaysInCycle: 30,
        },
        {
          propertyId: "prop-b",
          activeRooms: 5,
          unitPriceMinor: 30_000,
          cycle: "monthly",
          activeDays: 30,
          totalDaysInCycle: 30,
        },
      ],
    });

    expect(invoice.grossTotalMinor).toBe(450_000);
    expect(invoice.discountTotalMinor).toBe(90_000);
    expect(invoice.netTotalMinor).toBe(360_000);
    expect(Number.isInteger(invoice.netTotalMinor)).toBe(true);
  });
});
