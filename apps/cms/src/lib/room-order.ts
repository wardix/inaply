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

export interface RoomOrder {
  orderId: string;
  propertyId: string;
  roomNumber: string;
  totalMinor: number;
  status: "pending" | "preparing" | "delivered" | "cancelled";
}

export interface IptvChannel {
  channelId: string;
  propertyId: string;
  name: string;
  streamUrl: string;
  allowedPackages: string[];
}

export function createRoomOrderService(opts: {
  rooms: Map<string, OrderableRoom>;
  catalog: Map<string, CatalogItem>;
}) {
  let seq = 1;

  return {
    createOrder(input: {
      propertyId: string;
      roomNumber: string;
      items: OrderLineInput[];
    }): RoomOrder {
      const room = opts.rooms.get(`${input.propertyId}:${input.roomNumber}`);
      if (!room || room.status !== "occupied") {
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
        if (!item || !item.active) {
          throw new Error(`Item ${line.itemId} is unavailable`);
        }
        totalMinor += item.priceMinor * line.quantity;
      }

      return {
        orderId: `ord-${seq++}`,
        propertyId: input.propertyId,
        roomNumber: input.roomNumber,
        totalMinor,
        status: "pending",
      };
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
