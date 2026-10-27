# BM-DROID — licensing setup checkpoint

This is **not a complete application source repository or finished release**.

## Completed work saved here

- `supabase/migrations/20261027042501_init_licensing_schema.sql`: licensing/device tables, indexes, timestamp trigger, and deny-by-default RLS.
- `supabase/migrations/20261027042704_harden_function_search_path.sql`: trigger function search-path hardening.
- [Implementation status](docs/IMPLEMENTATION_STATUS.md): completed setup, remaining application work, and limitations.

These migrations are already applied to Supabase project `slyqunbridskbetksbov`. No database rows or secret keys are included. Do not manually reapply them to that project.

The application source baseline has not been modified for the requested licensing/kiosk features. The complete source ZIP, including the original PNG/ICO assets and these migrations, is provided separately in the project conversation. The GitHub integration rejected the full source upload because its request-confirmation limit is 65,536 bytes. Therefore this repository does not yet contain the application baseline or build files.

The encrypted-value columns only reserve ciphertext storage; encryption and licensing APIs are not implemented yet. No finished Windows EXE has been built or tested.
