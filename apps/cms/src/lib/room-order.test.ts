import { describe, expect, test } from "bun:test";
import { createRoomOrderService } from "./room-order";

describe("Room Service F&B Ordering", () => {
  test("requires occupied room, calculates server-side price ignoring client price, and rejects inactive items", () => {
    const svc = createRoomOrderService({
      rooms: new Map([
        ["prop-1:301", { propertyId: "prop-1", roomNumber: "301", status: "occupied" }],
        ["prop-1:302", { propertyId: "prop-1", roomNumber: "302", status: "vacant" }],
      ]),
      catalog: new Map([
        [
          "prop-1:nasi-goreng",
          {
            itemId: "nasi-goreng",
            propertyId: "prop-1",
            name: "Nasi Goreng",
            priceMinor: 85_000,
            active: true,
          },
        ],
        [
          "prop-1:sold-out",
          {
            itemId: "sold-out",
            propertyId: "prop-1",
            name: "Wagyu Steak",
            priceMinor: 350_000,
            active: false,
          },
        ],
      ]),
    });

    // Valid order on occupied room: client tries to send priceMinor=1, server must use 85_000 * 2 = 170_000
    const order = svc.createOrder({
      propertyId: "prop-1",
      roomNumber: "301",
      items: [{ itemId: "nasi-goreng", quantity: 2, clientPriceMinor: 1 }],
    });
    expect(order.totalMinor).toBe(170_000);
    expect(order.status).toBe("pending");

    // Vacant room -> rejected
    expect(() =>
      svc.createOrder({
        propertyId: "prop-1",
        roomNumber: "302",
        items: [{ itemId: "nasi-goreng", quantity: 1 }],
      })
    ).toThrow();

    // Inactive item -> rejected
    expect(() =>
      svc.createOrder({
        propertyId: "prop-1",
        roomNumber: "301",
        items: [{ itemId: "sold-out", quantity: 1 }],
      })
    ).toThrow();

    // Order status lifecycle & room/tenant isolation
    expect(svc.listOrdersForRoom("prop-1", "301")).toHaveLength(1);
    expect(svc.listOrdersForRoom("prop-1", "302")).toHaveLength(0);
    expect(svc.listOrdersForRoom("prop-2", "301")).toHaveLength(0);

    const preparing = svc.updateOrderStatus("prop-1", order.orderId, "preparing");
    expect(preparing.status).toBe("preparing");
    const delivered = svc.updateOrderStatus("prop-1", order.orderId, "delivered");
    expect(delivered.status).toBe("delivered");

    // Invalid transition (delivered -> cancelled) & cross-tenant update must throw
    expect(() => svc.updateOrderStatus("prop-1", order.orderId, "cancelled")).toThrow();
    expect(() => svc.updateOrderStatus("prop-2", order.orderId, "cancelled")).toThrow();
  });
});
