export type DeviceMode = "room_tv" | "signage";
export type DisplayOrientation = "landscape" | "portrait";
export type SignageZone = "lobby" | "lift" | "restaurant" | "pool" | string;

export interface SignageDisplay {
  deviceId: string;
  propertyId: string;
  mode: DeviceMode;
  zone: SignageZone;
  orientation: DisplayOrientation;
}

export interface SignageScheduleRule {
  propertyId: string;
  timezone: string;
  zone: SignageZone;
  startHour: number; // inclusive, 0..23
  endHour: number; // exclusive, 1..24
  playlistId: string;
}

export class SignagePairingForbiddenError extends Error {
  readonly status = 403;
  constructor(deviceId: string) {
    super(`Guest pairing is forbidden on signage device: ${deviceId}`);
    this.name = "SignagePairingForbiddenError";
  }
}

function getHourInTimezone(date: Date, timeZone: string): number {
  const formatted = new Intl.DateTimeFormat("en-US", {
    timeZone,
    hour: "numeric",
    hour12: false,
  }).format(date);
  const hour = Number.parseInt(formatted, 10);
  return hour === 24 ? 0 : hour;
}

export function createSignageService() {
  const displays = new Map<string, SignageDisplay>();
  const schedules: SignageScheduleRule[] = [];

  return {
    registerDisplay(display: SignageDisplay): SignageDisplay {
      displays.set(display.deviceId, { ...display });
      return { ...display };
    },

    pairGuest(deviceId: string, _guestId: string): { paired: true } {
      const display = displays.get(deviceId);
      if (!display) {
        throw new Error(`Display not found: ${deviceId}`);
      }
      if (display.mode === "signage") {
        throw new SignagePairingForbiddenError(deviceId);
      }
      return { paired: true };
    },

    addSchedule(rule: SignageScheduleRule): void {
      if (
        !Number.isInteger(rule.startHour) ||
        !Number.isInteger(rule.endHour) ||
        rule.startHour < 0 ||
        rule.endHour > 24 ||
        rule.startHour >= rule.endHour
      ) {
        throw new Error(
          `Invalid schedule hours: startHour=${rule.startHour}, endHour=${rule.endHour}`
        );
      }
      schedules.push({ ...rule });
    },

    resolveActivePlaylist(
      propertyId: string,
      zone: SignageZone,
      now: Date = new Date()
    ): string | null {
      for (const rule of schedules) {
        if (rule.propertyId !== propertyId || rule.zone !== zone) {
          continue;
        }
        const localHour = getHourInTimezone(now, rule.timezone);
        if (localHour >= rule.startHour && localHour < rule.endHour) {
          return rule.playlistId;
        }
      }
      return null;
    },
  };
}
