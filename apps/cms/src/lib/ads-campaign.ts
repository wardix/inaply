export type GuestSegment = "leisure" | "corporate" | "vip";

const ALLOWED_MEDIA_MIME_TYPES = new Set([
  "image/jpeg",
  "image/png",
  "image/webp",
  "video/mp4",
  "video/webm",
]);

export interface AdCampaign {
  propertyId: string;
  title: string;
  mediaUrl: string;
  mimeType: string;
  targetSegments: GuestSegment[];
  startsAt: Date;
  endsAt: Date;
  active: boolean;
}

export function createAdCampaign(input: Omit<AdCampaign, "active">): AdCampaign {
  if (input.endsAt.getTime() <= input.startsAt.getTime()) {
    throw new Error("ends_at must be greater than starts_at");
  }
  if (!ALLOWED_MEDIA_MIME_TYPES.has(input.mimeType.toLowerCase())) {
    throw new Error(`Unsupported media MIME type: ${input.mimeType}`);
  }
  return {
    ...input,
    active: true,
  };
}

export function filterActiveAdsForSegment(
  campaigns: AdCampaign[],
  segment: GuestSegment,
  now: Date = new Date()
): AdCampaign[] {
  const nowMs = now.getTime();
  return campaigns.filter((ad) => {
    if (!ad.active) return false;
    if (nowMs < ad.startsAt.getTime() || nowMs > ad.endsAt.getTime()) return false;
    if (ad.targetSegments.length === 0) return true;
    return ad.targetSegments.includes(segment);
  });
}
