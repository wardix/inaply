export interface AnonymousTelemetryEvent {
  property_id: string;
  protocol: "google_cast" | "mirroring" | string;
  session_duration: number;
  ad_id?: string;
  timestamp: string;
}

export interface RegisteredAd {
  ad_id: string;
  property_id: string;
  cta_url: string;
}

export interface AnalyticsServiceOptions {
  registeredAds?: Map<string, RegisteredAd>;
}

export interface DailyAnalyticsReport {
  property_id: string;
  date: string;
  total_sessions: number;
  protocol_counts: Record<string, number>;
  ad_impressions: number;
  peak_hour_utc: number | null;
}

export function createAnalyticsService(opts: AnalyticsServiceOptions = {}) {
  const registeredAds = opts.registeredAds ?? new Map<string, RegisteredAd>();
  const telemetryByProperty = new Map<string, AnonymousTelemetryEvent[]>();
  const qrClicksByPropertyAd = new Map<string, number>();

  return {
    ingestTelemetry(raw: Record<string, unknown>): { accepted: boolean } {
      const property_id = typeof raw.property_id === "string" ? raw.property_id : "";
      const protocol = typeof raw.protocol === "string" ? raw.protocol : "";
      const session_duration =
        typeof raw.session_duration === "number" && Number.isFinite(raw.session_duration)
          ? Math.max(0, raw.session_duration)
          : 0;
      const timestamp =
        typeof raw.timestamp === "string" && raw.timestamp
          ? raw.timestamp
          : new Date().toISOString();

      if (!property_id || !protocol) {
        return { accepted: false };
      }

      // Strictly whitelist only anonymous aggregate fields; drop any viewing PII.
      const sanitized: AnonymousTelemetryEvent = {
        property_id,
        protocol,
        session_duration,
        timestamp,
      };
      if (typeof raw.ad_id === "string" && raw.ad_id) {
        sanitized.ad_id = raw.ad_id;
      }

      const list = telemetryByProperty.get(property_id) ?? [];
      list.push(sanitized);
      telemetryByProperty.set(property_id, list);
      return { accepted: true };
    },

    getStoredTelemetry(propertyId: string): AnonymousTelemetryEvent[] {
      return [...(telemetryByProperty.get(propertyId) ?? [])];
    },

    trackQrRedirect(input: {
      ad_id: string;
      property_id: string;
      redirect_to?: string;
    }): { status: 302 | 404; location?: string } {
      const ad = registeredAds.get(input.ad_id);
      if (!ad || ad.property_id !== input.property_id) {
        return { status: 404 };
      }
      try {
        const parsed = new URL(ad.cta_url);
        if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
          return { status: 404 };
        }
      } catch {
        return { status: 404 };
      }
      const key = `${ad.property_id}:${ad.ad_id}`;
      qrClicksByPropertyAd.set(key, (qrClicksByPropertyAd.get(key) ?? 0) + 1);
      return {
        status: 302,
        location: ad.cta_url,
      };
    },

    getQrClicks(propertyId: string, adId: string): number {
      return qrClicksByPropertyAd.get(`${propertyId}:${adId}`) ?? 0;
    },

    getDailyReport(propertyId: string, date: string): DailyAnalyticsReport {
      const events = (telemetryByProperty.get(propertyId) ?? []).filter((e) =>
        e.timestamp.startsWith(date)
      );

      const protocol_counts: Record<string, number> = {};
      const hourCounts = new Map<number, number>();
      let ad_impressions = 0;

      for (const ev of events) {
        protocol_counts[ev.protocol] = (protocol_counts[ev.protocol] ?? 0) + 1;
        if (ev.ad_id) {
          ad_impressions++;
        }
        const hour = new Date(ev.timestamp).getUTCHours();
        if (!Number.isNaN(hour)) {
          hourCounts.set(hour, (hourCounts.get(hour) ?? 0) + 1);
        }
      }

      let peak_hour_utc: number | null = null;
      let maxCount = -1;
      for (const [hour, count] of hourCounts.entries()) {
        if (count > maxCount) {
          maxCount = count;
          peak_hour_utc = hour;
        }
      }

      return {
        property_id: propertyId,
        date,
        total_sessions: events.length,
        protocol_counts,
        ad_impressions,
        peak_hour_utc,
      };
    },
  };
}
