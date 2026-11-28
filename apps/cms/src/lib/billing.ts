export type BillingCycle = "monthly" | "annual";

export interface MeteringInput {
  activeRooms: number;
  unitPriceMinor: number;
  cycle: BillingCycle;
  activeDays: number;
  totalDaysInCycle: number;
}

export interface PropertyLineInput extends MeteringInput {
  propertyId: string;
}

export interface PartnerInvoiceInput {
  partnerId: string;
  period: string;
  discountBps: number; // basis points: 2000 = 20%
  properties: PropertyLineInput[];
}

export interface PartnerInvoice {
  invoiceId: string;
  partnerId: string;
  period: string;
  grossTotalMinor: number;
  discountTotalMinor: number;
  netTotalMinor: number;
}

export function calculateMeteredAmount(input: MeteringInput): number {
  if (
    input.activeRooms <= 0 ||
    input.unitPriceMinor <= 0 ||
    input.activeDays <= 0 ||
    input.totalDaysInCycle <= 0
  ) {
    return 0;
  }
  const monthsMultiplier = input.cycle === "annual" ? 12 : 1;
  const baseTotal = input.activeRooms * input.unitPriceMinor * monthsMultiplier;
  const clampedDays = Math.min(input.activeDays, input.totalDaysInCycle);
  return Math.round((baseTotal * clampedDays) / input.totalDaysInCycle);
}

export function createBillingEngine() {
  const invoicesByKey = new Map<string, PartnerInvoice>();

  return {
    generatePartnerInvoice(input: PartnerInvoiceInput): PartnerInvoice {
      const idempotencyKey = `${input.partnerId}:${input.period}`;
      const existing = invoicesByKey.get(idempotencyKey);
      if (existing) {
        return existing;
      }

      let grossTotalMinor = 0;
      for (const prop of input.properties) {
        grossTotalMinor += calculateMeteredAmount(prop);
      }

      const clampedBps = Math.max(0, Math.min(10_000, input.discountBps));
      const discountTotalMinor = Math.round((grossTotalMinor * clampedBps) / 10_000);
      const netTotalMinor = grossTotalMinor - discountTotalMinor;

      const invoice: PartnerInvoice = {
        invoiceId: `inv-${input.partnerId}-${input.period}`,
        partnerId: input.partnerId,
        period: input.period,
        grossTotalMinor,
        discountTotalMinor,
        netTotalMinor,
      };
      invoicesByKey.set(idempotencyKey, invoice);
      return invoice;
    },

    listInvoices(partnerId: string): PartnerInvoice[] {
      const result: PartnerInvoice[] = [];
      for (const inv of invoicesByKey.values()) {
        if (inv.partnerId === partnerId) {
          result.push(inv);
        }
      }
      return result;
    },
  };
}
