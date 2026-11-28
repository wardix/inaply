export interface WelcomeBrandingInput {
  hotelName: string;
  themeColor?: string;
  logoUrl?: string;
  welcomeTitle?: string;
  welcomeSubtitle?: string;
}

export interface WelcomeBranding {
  hotelName: string;
  themeColor: string;
  logoUrl: string;
  welcomeTitle: string;
  welcomeSubtitle: string;
}

const HEX_COLOR_RE = /^#[0-9a-fA-F]{6}$/;

function sanitizeHttpUrl(raw?: string): string {
  const trimmed = (raw ?? "").trim();
  if (!trimmed) return "";
  try {
    const parsed = new URL(trimmed);
    if (parsed.protocol === "https:" || parsed.protocol === "http:") {
      return parsed.toString();
    }
  } catch {
    // invalid URL
  }
  return "";
}

export function sanitizeWelcomeBranding(input: WelcomeBrandingInput): WelcomeBranding {
  const rawColor = (input.themeColor ?? "").trim();
  const themeColor = HEX_COLOR_RE.test(rawColor) ? rawColor.toUpperCase() : "#1E293B";

  const welcomeTitle =
    (input.welcomeTitle ?? "").trim() || `Welcome to ${input.hotelName}`;
  const welcomeSubtitle =
    (input.welcomeSubtitle ?? "").trim() || "Connect to Wi-Fi to cast from your device";

  return {
    hotelName: input.hotelName,
    themeColor,
    logoUrl: sanitizeHttpUrl(input.logoUrl),
    welcomeTitle,
    welcomeSubtitle,
  };
}
