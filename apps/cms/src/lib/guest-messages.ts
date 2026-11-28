export type MessagePriority = "normal" | "urgent" | "emergency";

export interface SendGuestMessageInput {
  propertyId: string;
  roomId: string | null;
  title: string;
  body: string;
  priority: MessagePriority;
  showAsOverlay: boolean;
}

export interface GuestMessageRecord extends SendGuestMessageInput {
  isBroadcast: boolean;
}

export function sendGuestMessage(
  input: SendGuestMessageInput,
  roomPropertyMap: Map<string, string>
): { status: 200; message: GuestMessageRecord } | { status: 403; error: string } {
  if (input.roomId !== null) {
    const ownerPropertyId = roomPropertyMap.get(input.roomId);
    if (!ownerPropertyId || ownerPropertyId !== input.propertyId) {
      return { status: 403, error: "Forbidden: room does not belong to target property" };
    }
  }

  return {
    status: 200,
    message: {
      ...input,
      isBroadcast: input.roomId === null,
    },
  };
}
