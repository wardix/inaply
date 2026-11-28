import { describe, expect, test } from "bun:test";
import { createBillingEngine } from "./billing";

describe("Invoice Idempotency", () => {
  test("re-running billing job for the same partner and period does not create duplicate invoices", () => {
    const engine = createBillingEngine();

    const input = {
      partnerId: "partner-isp-1",
      period: "2026-11",
      discountBps: 1000,
      properties: [
        {
          propertyId: "prop-a",
          activeRooms: 10,
          unitPriceMinor: 25_000,
          cycle: "monthly" as const,
          activeDays: 30,
          totalDaysInCycle: 30,
        },
      ],
    };

    const first = engine.generatePartnerInvoice(input);
    const second = engine.generatePartnerInvoice(input);

    expect(first.invoiceId).toBe(second.invoiceId);
    expect(engine.listInvoices("partner-isp-1")).toHaveLength(1);
  });
});
