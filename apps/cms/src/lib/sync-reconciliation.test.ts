import { describe, expect, test } from "bun:test";
import { SyncReconciler } from "./sync-reconciliation";

describe("Cloud-to-Edge Sync Reconciliation", () => {
  test("reconciles delta ads, messages, and force_checkout commands idempotently on reconnect", () => {
    const reconciler = new SyncReconciler();

    reconciler.enqueueDelta("prop-a", {
      seq: 1,
      ads: [{ id: "ad-1", title: "Spa" }],
      messages: [{ id: "msg-1", title: "Welcome" }],
      forceCheckoutRooms: ["301"],
    });
    reconciler.enqueueDelta("prop-a", {
      seq: 2,
      ads: [{ id: "ad-1", title: "Spa" }, { id: "ad-2", title: "Dinner" }],
      messages: [{ id: "msg-1", title: "Welcome" }],
      forceCheckoutRooms: ["301", "302"],
    });

    // Gateway reconnects with lastAckSeq = 0 -> receives seq 1..2 deduplicated
    const batch1 = reconciler.pullSince("prop-a", 0);
    expect(batch1.latestSeq).toBe(2);
    expect(batch1.ads.map((a) => a.id)).toEqual(["ad-1", "ad-2"]);
    expect(batch1.messages.map((m) => m.id)).toEqual(["msg-1"]);
    expect(batch1.forceCheckoutRooms).toEqual(["301", "302"]);

    // Re-pulling with lastAckSeq = 2 -> empty idempotent delta
    const batch2 = reconciler.pullSince("prop-a", 2);
    expect(batch2.ads).toEqual([]);
    expect(batch2.forceCheckoutRooms).toEqual([]);
  });
});
