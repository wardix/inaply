export type UserRole =
  | "super_admin"
  | "partner_admin"
  | "hotel_gm"
  | "hotel_marketing"
  | "hotel_fo"
  | "noc_support";

export interface AuthUser {
  id: string;
  role: UserRole;
  partnerId?: string;
  propertyId?: string;
}

export interface PropertyRecord {
  id: string;
  partnerId: string;
  name: string;
}

export function authorizePropertyAccess(
  user: AuthUser,
  property: PropertyRecord
): { status: 200 | 403 } {
  if (user.role === "super_admin" || user.role === "noc_support") {
    return { status: 200 };
  }
  if (user.role === "partner_admin") {
    return user.partnerId && user.partnerId === property.partnerId
      ? { status: 200 }
      : { status: 403 };
  }
  return user.propertyId && user.propertyId === property.id
    ? { status: 200 }
    : { status: 403 };
}

export function filterVisibleProperties(
  user: AuthUser,
  properties: PropertyRecord[]
): PropertyRecord[] {
  return properties.filter((p) => authorizePropertyAccess(user, p).status === 200);
}
