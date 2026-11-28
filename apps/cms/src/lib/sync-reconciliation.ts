export interface SyncDelta {
  seq: number;
  ads: Array<{ id: string; title: string }>;
  messages: Array<{ id: string; title: string }>;
  forceCheckoutRooms: string[];
}

export class SyncReconciler {
  private deltasByProperty = new Map<string, SyncDelta[]>();

  enqueueDelta(propertyId: string, delta: SyncDelta): void {
    const list = this.deltasByProperty.get(propertyId) ?? [];
    list.push(delta);
    this.deltasByProperty.set(propertyId, list);
  }

  pullSince(
    propertyId: string,
    lastAckSeq: number
  ): {
    latestSeq: number;
    ads: Array<{ id: string; title: string }>;
    messages: Array<{ id: string; title: string }>;
    forceCheckoutRooms: string[];
  } {
    const pending = (this.deltasByProperty.get(propertyId) ?? []).filter(
      (d) => d.seq > lastAckSeq
    );

    const adsById = new Map<string, { id: string; title: string }>();
    const msgsById = new Map<string, { id: string; title: string }>();
    const rooms = new Set<string>();
    let latestSeq = lastAckSeq;

    for (const d of pending) {
      if (d.seq > latestSeq) latestSeq = d.seq;
      for (const ad of d.ads) adsById.set(ad.id, ad);
      for (const msg of d.messages) msgsById.set(msg.id, msg);
      for (const room of d.forceCheckoutRooms) rooms.add(room);
    }

    return {
      latestSeq,
      ads: Array.from(adsById.values()),
      messages: Array.from(msgsById.values()),
      forceCheckoutRooms: Array.from(rooms),
    };
  }
}
