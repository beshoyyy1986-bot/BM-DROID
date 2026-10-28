# APK Box

Admin panel -> pick APK + duration -> export a licensed Windows EXE that runs only that APK inside a minimal Android 9 (Android-x86) VM.

1. Licensing backend: Supabase edge function `bm-licensing` (already deployed).
2. Copy `apkbox/ci/runtime.yml` to `.github/workflows/runtime.yml` and run it from the Actions tab. It builds `runtime.zip` (QEMU + minimal Android 9) and `launcher.exe`, smoke-tests the boot with KVM and publishes them in release `runtime-v1`.
3. Admin: download `launcher.exe` from the release into `apkbox/admin/stub/`, copy `admin/.env.example` to `admin/.env` (set `ADMIN_TOKEN`, `RUNTIME_SHA256` from `runtime.zip.sha256`), run `node apkbox/admin/server.js`, open http://localhost:8080.
