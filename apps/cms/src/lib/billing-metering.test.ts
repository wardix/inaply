import { describe, expect, test } from "bun:test";
import { calculateMeteredAmount } from "./billing";

describe("Billing Metering", () => {
  test("calculates monthly active rooms with proration and annual cycle", () => {
    // 10 rooms at Rp 25.000/room/month, active for 15 of 30 days -> 125.000
    const prorated = calculateMeteredAmount({
      activeRooms: 10,
      unitPriceMinor: 25_000,
      cycle: "monthly",
      activeDays: 15,
      totalDaysInCycle: 30,
    });
    expect(prorated).toBe(125_000);

    // Annual cycle: 10 rooms * 25.000 * 12 months = 3.000.000
    const annual = calculateMeteredAmount({
      activeRooms: 10,
      unitPriceMinor: 25_000,
      cycle: "annual",
      activeDays: 365,
      totalDaysInCycle: 365,
    });
    expect(annual).toBe(3_000_000);
  });
});
