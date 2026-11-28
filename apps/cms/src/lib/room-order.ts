export interface OrderableRoom {
  propertyId: string;
  roomNumber: string;
  status: "vacant" | "occupied";
}

export interface CatalogItem {
  itemId: string;
  propertyId: string;
  name: string;
  priceMinor: number;
  active: boolean;
}

export interface OrderLineInput {
  itemId: string;
  quantity: number;
  clientPriceMinor?: number; // ignored; server-side price is authoritative
}

export type OrderStatus = "pending" | "preparing" | "delivered" | "cancelled";

export interface RoomOrder {
  orderId: string;
  propertyId: string;
  roomNumber: string;
  totalMinor: number;
  status: OrderStatus;
}

export interface IptvChannel {
  channelId: string;
  propertyId: string;
  name: string;
  streamUrl: string;
  allowedPackages: string[];
}

const VALID_TRANSITIONS: Record<OrderStatus, Set<OrderStatus>> = {
  pending: new Set(["preparing", "cancelled"]),
  preparing: new Set(["delivered", "cancelled"]),
  delivered: new Set(),
  cancelled: new Set(),
};

export function createRoomOrderService(opts: {
  rooms: Map<string, OrderableRoom>;
  catalog: Map<string, CatalogItem>;
}) {
  let seq = 1;
  const orders = new Map<string, RoomOrder>();

  return {
    createOrder(input: {
      propertyId: string;
      roomNumber: string;
      items: OrderLineInput[];
    }): RoomOrder {
      const room = opts.rooms.get(`${input.propertyId}:${input.roomNumber}`);
      if (!room || room.propertyId !== input.propertyId || room.status !== "occupied") {
        throw new Error(`Room ${input.roomNumber} is not occupied`);
      }
      if (input.items.length === 0) {
        throw new Error("Order must contain at least one item");
      }

      let totalMinor = 0;
      for (const line of input.items) {
        if (!Number.isInteger(line.quantity) || line.quantity <= 0) {
          throw new Error(`Invalid quantity for item ${line.itemId}`);
        }
        const item = opts.catalog.get(`${input.propertyId}:${line.itemId}`);
        if (
          !item ||
          item.propertyId !== input.propertyId ||
          !item.active ||
          !Number.isInteger(item.priceMinor) ||
          item.priceMinor < 0
        ) {
          throw new Error(`Item ${line.itemId} is unavailable`);
        }
        totalMinor += item.priceMinor * line.quantity;
      }

      const order: RoomOrder = {
        orderId: `ord-${seq++}`,
        propertyId: input.propertyId,
        roomNumber: input.roomNumber,
        totalMinor,
        status: "pending",
      };
      orders.set(order.orderId, order);
      return { ...order };
    },

    listOrdersForRoom(propertyId: string, roomNumber: string): RoomOrder[] {
      const result: RoomOrder[] = [];
      for (const order of orders.values()) {
        if (order.propertyId === propertyId && order.roomNumber === roomNumber) {
          result.push({ ...order });
        }
      }
      return result;
    },

    updateOrderStatus(
      propertyId: string,
      orderId: string,
      nextStatus: OrderStatus
    ): RoomOrder {
      const order = orders.get(orderId);
      if (!order || order.propertyId !== propertyId) {
        throw new Error(`Order not found: ${orderId}`);
      }
      if (!VALID_TRANSITIONS[order.status].has(nextStatus)) {
        throw new Error(`Invalid order status transition: ${order.status} -> ${nextStatus}`);
      }
      order.status = nextStatus;
      return { ...order };
    },
  };
}

export function filterIptvChannels(
  channels: IptvChannel[],
  propertyId: string,
  roomPackage: string
): IptvChannel[] {
  return channels.filter(
    (ch) => ch.propertyId === propertyId && ch.allowedPackages.includes(roomPackage)
  );
}
