import { describe, expect, test } from "bun:test";
import { authorizePropertyAccess, filterVisibleProperties, type AuthUser, type PropertyRecord } from "./rbac";

describe("Multi-tenant RBAC", () => {
  const properties: PropertyRecord[] = [
    { id: "prop-a", partnerId: "partner-1", name: "Hotel A" },
    { id: "prop-b", partnerId: "partner-2", name: "Hotel B" },
    { id: "prop-c", partnerId: "partner-1", name: "Hotel C" },
  ];

  test("hotel_fo / hotel_gm from Property A always gets 403 when accessing Property B", () => {
    const foUser: AuthUser = { id: "u1", role: "hotel_fo", propertyId: "prop-a" };
    const gmUser: AuthUser = { id: "u2", role: "hotel_gm", propertyId: "prop-a" };

    expect(authorizePropertyAccess(foUser, properties[0]).status).toBe(200);
    expect(authorizePropertyAccess(foUser, properties[1]).status).toBe(403);
    expect(authorizePropertyAccess(gmUser, properties[1]).status).toBe(403);
  });

  test("partner_admin only sees properties under their partner", () => {
    const partnerAdmin: AuthUser = { id: "u3", role: "partner_admin", partnerId: "partner-1" };
    const visible = filterVisibleProperties(partnerAdmin, properties);
    expect(visible.map((p) => p.id)).toEqual(["prop-a", "prop-c"]);
    expect(authorizePropertyAccess(partnerAdmin, properties[1]).status).toBe(403);
  });
});
