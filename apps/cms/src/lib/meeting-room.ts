export interface MeetingSession {
  propertyId: string;
  meetingRoomId: string;
  sessionCode: string;
  expiresAt: Date;
  backdropTitle: string;
  presenters: Set<string>;
}

export interface MeetingRoomServiceOptions {
  onDisconnect?: (presenterId: string, meetingRoomId: string) => void;
}

export function createMeetingRoomService(opts: MeetingRoomServiceOptions = {}) {
  const sessions = new Map<string, MeetingSession>();
  const key = (propertyId: string, meetingRoomId: string) => `${propertyId}:${meetingRoomId}`;

  return {
    startMeetingSession(input: {
      propertyId: string;
      meetingRoomId: string;
      sessionCode: string;
      expiresAt: Date;
      backdropTitle: string;
    }): MeetingSession {
      const code = input.sessionCode.trim();
      if (!code) {
        throw new Error("sessionCode must not be empty");
      }
      if (Number.isNaN(input.expiresAt.getTime())) {
        throw new Error("expiresAt must be a valid Date");
      }

      const k = key(input.propertyId, input.meetingRoomId);
      const prev = sessions.get(k);
      if (prev) {
        for (const p of prev.presenters) {
          opts.onDisconnect?.(p, input.meetingRoomId);
        }
        prev.presenters.clear();
      }

      const session: MeetingSession = {
        ...input,
        sessionCode: code,
        presenters: new Set<string>(),
      };
      sessions.set(k, session);
      return session;
    },

    pairPresenter(
      propertyId: string,
      meetingRoomId: string,
      sessionCode: string,
      presenterId: string,
      now: Date = new Date()
    ): void {
      const session = sessions.get(key(propertyId, meetingRoomId));
      if (!session || session.sessionCode !== sessionCode) {
        throw new Error("Invalid meeting room or session code");
      }
      if (now.getTime() >= session.expiresAt.getTime()) {
        throw new Error("Meeting session has expired");
      }
      session.presenters.add(presenterId);
    },

    getActivePresenters(
      propertyId: string,
      meetingRoomId: string,
      now: Date = new Date()
    ): string[] {
      const session = sessions.get(key(propertyId, meetingRoomId));
      if (!session || now.getTime() >= session.expiresAt.getTime()) {
        return [];
      }
      return [...session.presenters];
    },

    getSessionBackdrop(
      propertyId: string,
      meetingRoomId: string,
      now: Date = new Date()
    ): string | null {
      const session = sessions.get(key(propertyId, meetingRoomId));
      if (!session || now.getTime() >= session.expiresAt.getTime()) {
        return null;
      }
      return session.backdropTitle;
    },

    updateBackdrop(
      propertyId: string,
      meetingRoomId: string,
      backdropTitle: string
    ): void {
      const session = sessions.get(key(propertyId, meetingRoomId));
      if (!session) {
        throw new Error("Meeting session not found");
      }
      session.backdropTitle = backdropTitle;
    },

    expireStale(now: Date = new Date()): number {
      let expired = 0;
      for (const [k, session] of sessions.entries()) {
        if (now.getTime() >= session.expiresAt.getTime()) {
          for (const p of session.presenters) {
            opts.onDisconnect?.(p, session.meetingRoomId);
          }
          session.presenters.clear();
          sessions.delete(k);
          expired++;
        }
      }
      return expired;
    },
  };
}
