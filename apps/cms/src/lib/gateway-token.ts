import { createHash, randomBytes } from "crypto";

interface StoredToken {
  propertyId: string;
  tokenHash: string;
  expiresAtMs: number;
  used: boolean;
}

export class GatewayTokenService {
  private tokensByHash = new Map<string, StoredToken>();

  constructor(private readonly nowMs: () => number = () => Date.now()) {}

  private hashToken(rawToken: string): string {
    return createHash("sha256").update(rawToken).digest("hex");
  }

  issueToken(
    propertyId: string,
    ttlMs: number
  ): { rawToken: string; tokenHash: string; expiresAtMs: number } {
    const rawToken = randomBytes(24).toString("hex");
    const tokenHash = this.hashToken(rawToken);
    const expiresAtMs = this.nowMs() + ttlMs;
    this.tokensByHash.set(tokenHash, {
      propertyId,
      tokenHash,
      expiresAtMs,
      used: false,
    });
    return { rawToken, tokenHash, expiresAtMs };
  }

  consumeToken(
    rawToken: string
  ): { valid: true; propertyId: string } | { valid: false; reason: "not_found" | "already_used" | "expired" } {
    const tokenHash = this.hashToken(rawToken);
    const entry = this.tokensByHash.get(tokenHash);
    if (!entry) {
      return { valid: false, reason: "not_found" };
    }
    if (entry.used) {
      return { valid: false, reason: "already_used" };
    }
    if (this.nowMs() > entry.expiresAtMs) {
      return { valid: false, reason: "expired" };
    }
    entry.used = true;
    return { valid: true, propertyId: entry.propertyId };
  }
}
