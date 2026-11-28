export interface Property {
  id: string;
  partnerId: string;
  name: string;
}

export interface Gateway {
  id: string;
  propertyId: string;
  name: string;
}

export interface Room {
  id: string;
  propertyId: string;
  roomNumber: string;
}

export interface Receiver {
  id: string;
  gatewayId: string;
  roomId: string;
  deviceId: string;
}

export class OnboardingStore {
  private seq = 0;
  private properties = new Map<string, Property>();
  private gateways = new Map<string, Gateway>();
  private rooms = new Map<string, Room>();
  private receivers = new Map<string, Receiver>();
  private roomKeys = new Set<string>(); // `${propertyId}:${roomNumber}`
  private deviceIds = new Set<string>();

  private nextId(prefix: string): string {
    this.seq += 1;
    return `${prefix}-${this.seq}`;
  }

  createProperty(input: { partnerId: string; name: string }): Property {
    const prop: Property = { id: this.nextId("prop"), ...input };
    this.properties.set(prop.id, prop);
    return prop;
  }

  createGateway(input: { propertyId: string; name: string }): Gateway {
    if (!this.properties.has(input.propertyId)) {
      throw new Error(`Unknown property ${input.propertyId}`);
    }
    const gw: Gateway = { id: this.nextId("gw"), ...input };
    this.gateways.set(gw.id, gw);
    return gw;
  }

  createRoom(input: { propertyId: string; roomNumber: string }): Room {
    if (!this.properties.has(input.propertyId)) {
      throw new Error(`Unknown property ${input.propertyId}`);
    }
    const key = `${input.propertyId}:${input.roomNumber}`;
    if (this.roomKeys.has(key)) {
      throw new Error(`Duplicate room_number ${input.roomNumber} in property ${input.propertyId}`);
    }
    this.roomKeys.add(key);
    const room: Room = { id: this.nextId("room"), ...input };
    this.rooms.set(room.id, room);
    return room;
  }

  createReceiver(input: { gatewayId: string; roomId: string; deviceId: string }): Receiver {
    const gw = this.gateways.get(input.gatewayId);
    if (!gw) {
      throw new Error(`Unknown gateway ${input.gatewayId}`);
    }
    const room = this.rooms.get(input.roomId);
    if (!room) {
      throw new Error(`Unknown room ${input.roomId}`);
    }
    if (gw.propertyId !== room.propertyId) {
      throw new Error(`Cross-property mismatch: gateway ${gw.id} and room ${room.id} belong to different properties`);
    }
    if (this.deviceIds.has(input.deviceId)) {
      throw new Error(`Duplicate global device_id ${input.deviceId}`);
    }
    this.deviceIds.add(input.deviceId);
    const rcv: Receiver = { id: this.nextId("rcv"), ...input };
    this.receivers.set(rcv.id, rcv);
    return rcv;
  }
}
