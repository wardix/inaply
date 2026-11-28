-- Inaply Multi-Tenant Cloud CMS Schema (PostgreSQL 15+)
-- Hierarchy: Partners (ISP/SI) -> Properties (Hotels) -> Gateways -> Rooms -> Receivers

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE partners (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    support_email   TEXT NOT NULL,
    support_phone   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE properties (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partner_id          UUID REFERENCES partners(id) ON DELETE SET NULL,
    name                TEXT NOT NULL,
    code                TEXT NOT NULL UNIQUE,
    timezone            TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    wifi_ssid           TEXT NOT NULL,
    default_checkout_hr SMALLINT NOT NULL DEFAULT 12,
    pms_provider        TEXT NOT NULL DEFAULT 'none', -- none | vhp | cloudbeds | opera_ohip | custom_webhook
    pms_config_enc      JSONB NOT NULL DEFAULT '{}'::jsonb,
    branding_config     JSONB NOT NULL DEFAULT '{}'::jsonb, -- logo_url, primary_color, welcome_title
    status              TEXT NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE gateways (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id         UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    serial_number       TEXT NOT NULL UNIQUE,
    agent_version       TEXT NOT NULL,
    lan_ip              INET,
    pairing_mode        TEXT NOT NULL DEFAULT 'hybrid', -- hybrid | portal_only | qr_pin_only
    last_heartbeat_at   TIMESTAMPTZ,
    edge_status         TEXT NOT NULL DEFAULT 'offline',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rooms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    room_number     TEXT NOT NULL,
    floor           TEXT,
    segment         TEXT NOT NULL DEFAULT 'leisure', -- leisure | corporate | vip
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (property_id, room_number)
);

CREATE TABLE receivers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id         UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    room_id             UUID REFERENCES rooms(id) ON DELETE CASCADE,
    mode                TEXT NOT NULL DEFAULT 'room_tv',
    zone                TEXT,
    orientation         TEXT NOT NULL DEFAULT 'landscape',
    device_id           TEXT NOT NULL UNIQUE,
    label               TEXT NOT NULL DEFAULT 'Main TV',
    lan_ip              INET NOT NULL,
    mac_address         MACADDR,
    hardware_model      TEXT NOT NULL DEFAULT 'certified-google-tv',
    app_version         TEXT,
    device_owner_active BOOLEAN NOT NULL DEFAULT false,
    hdmi_cec_enabled    BOOLEAN NOT NULL DEFAULT true,
    last_seen_at        TIMESTAMPTZ,
    status              TEXT NOT NULL DEFAULT 'offline',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE guest_stays (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    room_id         UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    guest_name      TEXT NOT NULL,
    segment         TEXT NOT NULL DEFAULT 'leisure',
    pms_resv_id     TEXT,
    checked_in_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expected_out_at TIMESTAMPTZ NOT NULL,
    checked_out_at  TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'checked_in' -- checked_in | checked_out
);

CREATE TABLE ad_campaigns (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    media_type      TEXT NOT NULL DEFAULT 'image', -- image | video
    media_url       TEXT NOT NULL,
    media_sha256    TEXT,
    cta_url         TEXT,
    target_segments TEXT[] NOT NULL DEFAULT ARRAY['leisure', 'corporate', 'vip'],
    duration_sec    SMALLINT NOT NULL DEFAULT 15,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    starts_at       TIMESTAMPTZ,
    ends_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE guest_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    room_id         UUID REFERENCES rooms(id) ON DELETE CASCADE, -- NULL means broadcast to property
    title           TEXT NOT NULL,
    body            TEXT NOT NULL,
    priority        TEXT NOT NULL DEFAULT 'normal', -- normal | urgent | emergency
    show_as_overlay BOOLEAN NOT NULL DEFAULT true,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE billing_subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    plan_code       TEXT NOT NULL DEFAULT 'managed_opex_room',
    active_rooms    INT NOT NULL,
    price_per_room  NUMERIC(12,2) NOT NULL,
    currency        TEXT NOT NULL DEFAULT 'IDR',
    billing_cycle   TEXT NOT NULL DEFAULT 'monthly', -- monthly | annual
    next_invoice_at DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_rooms_property ON rooms(property_id);
CREATE INDEX idx_receivers_room ON receivers(room_id);
CREATE INDEX idx_stays_active ON guest_stays(property_id, room_id) WHERE status = 'checked_in';
CREATE INDEX idx_ads_property_active ON ad_campaigns(property_id) WHERE is_active = true;
