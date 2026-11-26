export type PairingMode = "portal" | "radius" | "qr_pin" | "fo_manual";
export type GuestSegment = "leisure" | "corporate" | "vip";
export type MessagePriority = "normal" | "urgent" | "emergency";

export interface PairResult {
  session_id: string;
  room_number: string;
  guest_ip: string;
  pairing_mode: PairingMode;
  receivers: Array<{
    device_id: string;
    label: string;
    ip: string;
  }>;
  expires_at: string;
}

export interface AdItem {
  id: string;
  title: string;
  media_url: string;
  cta_url?: string;
  duration_sec: number;
}

export interface GuestMessageItem {
  id: string;
  title: string;
  body: string;
  priority: MessagePriority;
  show_as_overlay: boolean;
}

export interface TVBackdropState {
  room_number: string;
  device_id: string;
  device_label: string;
  occupied: boolean;
  guest_name?: string;
  segment: GuestSegment;
  hotel_name: string;
  welcome_title: string;
  welcome_subtitle: string;
  wifi_ssid: string;
  pairing_pin: string;
  pairing_url: string;
  active_paired_devices: number;
  reset_epoch: number;
  ads: AdItem[];
  messages: GuestMessageItem[];
}
