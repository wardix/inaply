import { timingSafeEqual } from "crypto";

export interface GatewaySyncCredential {
  propertyId: string;
  token: string;
}

export function authenticateGatewaySync(
  gatewayId: string,
  propertyId: string,
  token: string,
  registry: Map<string, GatewaySyncCredential>
): { status: 200 } | { status: 401; error: string } {
  const cred = registry.get(gatewayId);
  if (!cred || !token || cred.propertyId !== propertyId) {
    return { status: 401, error: "Unauthorized" };
  }

  const a = Buffer.from(token);
  const b = Buffer.from(cred.token);
  if (a.length !== b.length || !timingSafeEqual(a, b)) {
    return { status: 401, error: "Unauthorized" };
  }
  return { status: 200 };
}
