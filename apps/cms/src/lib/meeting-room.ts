export interface MeetingSession {
  propertyId: string;
  meetingRoomId: string;
  sessionCode: string;
  expiresAt: Date;
  backdropTitle: string;
  presenters: Set<string>;
}

export function createMeetingRoomService() {
  const sessions = new Map<string, MeetingSession>();

  return {
    startMeetingSession(input: {
      propertyId: string;
      meetingRoomId: string;
      sessionCode: string;
      expiresAt: Date;
      backdropTitle: string;
    }): MeetingSession {
      const session: MeetingSession = {
        ...input,
        presenters: new Set<string>(),
      };
      sessions.set(input.meetingRoomId, session);
      return session;
    },

    pairPresenter(
      meetingRoomId: string,
      sessionCode: string,
      presenterId: string,
      now: Date = new Date()
    ): void {
      const session = sessions.get(meetingRoomId);
      if (!session || session.sessionCode !== sessionCode) {
        throw new Error("Invalid meeting room or session code");
      }
      if (now.getTime() >= session.expiresAt.getTime()) {
        throw new Error("Meeting session has expired");
      }
      session.presenters.add(presenterId);
    },

    getActivePresenters(meetingRoomId: string): string[] {
      return [...(sessions.get(meetingRoomId)?.presenters ?? [])];
    },

    expireStale(now: Date = new Date()): number {
      let expired = 0;
      for (const [roomId, session] of sessions.entries()) {
        if (now.getTime() >= session.expiresAt.getTime()) {
          session.presenters.clear();
          sessions.delete(roomId);
          expired++;
        }
      }
      return expired;
    },
  };
}
