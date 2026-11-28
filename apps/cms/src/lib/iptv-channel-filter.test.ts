import { describe, expect, test } from "bun:test";
import { filterIptvChannels, type IptvChannel } from "./room-order";

describe("IPTV Channel Filter", () => {
  test("filters channels by property and room package without leaking stream URLs cross-tenant", () => {
    const channels: IptvChannel[] = [
      {
        channelId: "ch-1",
        propertyId: "prop-1",
        name: "News HD",
        streamUrl: "udp://239.1.1.1:5000",
        allowedPackages: ["basic", "vip"],
      },
      {
        channelId: "ch-2",
        propertyId: "prop-1",
        name: "Cinema VIP",
        streamUrl: "udp://239.1.1.2:5000",
        allowedPackages: ["vip"],
      },
      {
        channelId: "ch-3",
        propertyId: "prop-2",
        name: "Prop 2 Internal",
        streamUrl: "udp://239.2.2.1:5000",
        allowedPackages: ["basic", "vip"],
      },
    ];

    const basicProp1 = filterIptvChannels(channels, "prop-1", "basic");
    expect(basicProp1.map((c) => c.channelId)).toEqual(["ch-1"]);

    const vipProp1 = filterIptvChannels(channels, "prop-1", "vip");
    expect(vipProp1.map((c) => c.channelId)).toEqual(["ch-1", "ch-2"]);

    // Cross-tenant check: prop-2 channel must never appear for prop-1
    expect(vipProp1.some((c) => c.streamUrl.includes("239.2.2.1"))).toBe(false);
  });
});
