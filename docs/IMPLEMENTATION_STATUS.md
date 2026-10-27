# Implementation status

This repository is a source snapshot, not a completed licensing/kiosk release.

## Included

- The 67-file BM-Droid Lite source baseline supplied in BM-Droid.zip. No application code changes were made during the licensing setup.
- SQL copies of the two migrations already applied to Supabase project `slyqunbridskbetksbov`: licensing/device tables, indexes, RLS, timestamps, and trigger search-path hardening.
- The full downloadable source ZIP includes all 16 original PNG/ICO assets, without conversion.

## Not implemented or tested yet

- Android 5.1/API 22-only installer and the new lightweight application defaults.
- Removal of game controls, macros, gamepad code, and related dependencies.
- Owner APK/export workflow, subscriber kiosk, cookie handling, and three status LEDs.
- Proxy traffic redirection, protected DNS/UDP handling, or fail-closed enforcement. The existing proxy code must not be treated as leak-proof.
- License activation/validation/heartbeat Edge Functions or client integration.
- AES-256-GCM encryption/decryption implementation. `*_enc` columns only reserve ciphertext storage; they do not encrypt values themselves.
- Admin dashboard and Vercel deployment.
- Automated tests and a Windows EXE build for these requested features.

## Security and deployment

No database rows, subscriber data, cookies, proxy credentials, service-role keys, or encryption keys are included. RLS deliberately has no anon/authenticated policies. Future server-side code must keep privileged credentials and the encryption key in environment secrets, never in the desktop app or repository. There is no working licensing API yet.

Build the Windows application on Windows using the existing `build.ps1`. Consult `LITE.md` for the supplied baseline. The SQL migrations are already applied to the named Supabase project; do not reapply them manually there.

## Repository upload checkpoint

The GitHub repository currently contains only the completed licensing SQL migrations and status documentation. Application source and binary assets are supplied in the full source ZIP, not uploaded to this repository: the integration rejected the full upload because its confirmation payload exceeded 65,536 bytes. This must not be mistaken for a complete application repository.
