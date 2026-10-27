-- BM-Droid licensing schema (secrets encrypted at rest; key lives only in server env)
create extension if not exists pgcrypto;

create table public.licenses (
  id uuid primary key default gen_random_uuid(),
  license_key text unique not null default encode(gen_random_bytes(16), 'hex'),
  subscriber text not null default '',
  note text not null default '',
  app_name text not null default '',
  app_package text not null default '',
  app_apk_url text not null default '',
  app_apk_sha256 text not null default '',
  duration_days integer not null default 30,
  status text not null default 'active' check (status in ('active','paused','banned','expired')),
  activated_at timestamptz,
  expires_at timestamptz,
  self_destruct boolean not null default false,
  android_api text not null default '22',
  cores integer not null default 1,
  ram integer not null default 1024,
  width integer not null default 480,
  height integer not null default 800,
  density integer not null default 240,
  disk integer not null default 4,
  gpu text not null default 'host',
  eco boolean not null default true,
  proxy_enabled boolean not null default false,
  proxy_type text not null default 'http',
  proxy_host text not null default '',
  proxy_port integer not null default 0,
  proxy_user text not null default '',
  proxy_secret_enc text,
  proxy_uri_enc text,
  allow_proxy_edit boolean not null default true,
  allow_cookie_edit boolean not null default true,
  cookies_enc text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table public.devices (
  id uuid primary key default gen_random_uuid(),
  license_id uuid not null references public.licenses(id) on delete cascade,
  device_hash text not null,
  app_version text not null default '',
  exit_ip text not null default '',
  proxy_tcp boolean not null default false,
  proxy_udp_dns boolean not null default false,
  proxy_failclosed boolean not null default false,
  first_seen timestamptz not null default now(),
  last_seen timestamptz not null default now(),
  unique (license_id, device_hash)
);

create index devices_license_idx on public.devices(license_id);
create index devices_last_seen_idx on public.devices(last_seen desc);

create or replace function public.touch_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end $$;

create trigger licenses_touch before update on public.licenses
for each row execute function public.touch_updated_at();

-- Only service_role (edge functions + dashboard server) may read/write.
alter table public.licenses enable row level security;
alter table public.devices enable row level security;
-- No policies: anon/authenticated denied; service_role bypasses RLS.
-- *_enc columns reserve ciphertext storage. Encryption is not implemented by SQL.
