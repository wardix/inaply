export interface FleetDeviceHeartbeat {
  deviceId: string;
  propertyId: string;
  appVersion: string;
  hdmiCec: boolean;
  lastSeenMs: number;
}

export interface FleetDeviceStatus extends FleetDeviceHeartbeat {
  status: "online" | "offline";
}

export function computeFleetStatus(
  devices: FleetDeviceHeartbeat[],
  nowMs: number,
  offlineThresholdMs: number
): FleetDeviceStatus[] {
  return devices.map((d) => ({
    ...d,
    status: nowMs - d.lastSeenMs > offlineThresholdMs ? "offline" : "online",
  }));
}
