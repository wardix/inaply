import { describe, expect, test } from "bun:test";
import { sanitizeWelcomeBranding } from "./welcome-branding";

describe("Welcome Screen Branding", () => {
  test("sanitizes hex color and logo URL and applies hotel defaults when optional fields are empty", () => {
    const defaulted = sanitizeWelcomeBranding({
      hotelName: "Grand Inaply",
      themeColor: "javascript:alert(1)",
      logoUrl: "javascript:alert(1)",
      welcomeTitle: "",
      welcomeSubtitle: "",
    });

    expect(defaulted.themeColor).toBe("#1E293B");
    expect(defaulted.logoUrl).toBe("");
    expect(defaulted.welcomeTitle).toBe("Welcome to Grand Inaply");
    expect(defaulted.welcomeSubtitle).toBe("Connect to Wi-Fi to cast from your device");

    const custom = sanitizeWelcomeBranding({
      hotelName: "Grand Inaply",
      themeColor: "#0f172a",
      logoUrl: "https://cdn.inaply.local/logo.png",
      welcomeTitle: "Warmest Greetings",
      welcomeSubtitle: "Enjoy your stay",
    });

    expect(custom.themeColor).toBe("#0F172A");
    expect(custom.logoUrl).toBe("https://cdn.inaply.local/logo.png");
    expect(custom.welcomeTitle).toBe("Warmest Greetings");
  });
});
