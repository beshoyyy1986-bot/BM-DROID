#!/usr/bin/env bash
# Builds runtime.zip = portable QEMU for Windows + minimal Android-x86 9.0 (only what is needed to run one APK).
# Runs on a GitHub Actions ubuntu runner (see .github/workflows/runtime.yml).
set -euxo pipefail
trap 'status=$?; echo "::error::Build failed at line $LINENO (exit $status)"' ERR
ROOT=$PWD; W=$ROOT/work; rm -rf "$W"; mkdir -p "$W"; cd "$W"
sudo apt-get update -qq
sudo apt-get install -y -qq squashfs-tools e2fsprogs qemu-system-x86 qemu-utils imagemagick zip curl xz-utils
# Retry partial transfers and label failures so they can be diagnosed in Actions.
download() {
  local url="$1" out="$2" attempt
  for attempt in 1 2 3 4 5; do
    echo "Downloading $out (attempt $attempt/5)"
    if curl -fL --connect-timeout 30 --max-time 1800 --retry 3 --retry-all-errors --retry-delay 5 -C - -o "$out" "$url"; then return 0; fi
    sleep "$((attempt * 5))"
  done
  echo "::error::Download failed for $out after retries"
  return 1
}
download https://github.com/ip7z/7zip/releases/download/24.08/7z2408-linux-x64.tar.xz 7zip.tar.xz
tar -xJf 7zip.tar.xz 7zz
download "https://sourceforge.net/projects/android-x86/files/Release%209.0/android-x86_64-9.0-r2.iso/download" a.iso
./7zz x -y a.iso -oiso >/dev/null
unsquashfs -d sfs iso/system.sfs >/dev/null
mkdir sys && sudo mount -o loop sfs/system.img sys

# ---- strip everything that is not needed to run one APK ----
for a in WallpaperBackup Camera2 Development GoogleExtShared Traceur GoogleContactsSyncAdapter TermOnePlus BluetoothMidiService Bluetooth BasicSmsReceiver SimAppDialog Calendar DeskClock ExactCalculator PrintSpooler PartnerBookmarksProvider BookmarkProvider Gallery2 BasicDreams WallpaperPicker HTMLViewer GoogleCalendarSyncAdapter BuiltInPrintService Gmail2 LiveWallpapersPicker GooglePrintRecommendationService EasterEgg RSSReader Chrome CarrierDefaultApp CompanionDeviceManager NotePad; do sudo rm -rf sys/app/$a; done
for p in GoogleLoginService ManagedProvisioning GmsCoreSetup Eleven Dialer Contacts GmsCore GoogleOneTimeInitializer SetupWizard GoogleFeedback GoogleExtServices GoogleBackupTransport Velvet GoogleServicesFramework GoogleRestore Phonesky GooglePartnerSetup AnalyticsService EmergencyInfo SettingsIntelligence CalendarProvider GooglePackageInstaller Taskbar WallpaperCropper StatementService; do sudo rm -rf sys/priv-app/$p; done

# ---- boot hook: install the APK from the virtual drive and keep it running full-screen ----
sudo cp "$ROOT/apkbox/image/apkbox.sh" sys/bin/apkbox.sh; sudo chmod 755 sys/bin/apkbox.sh
printf 'service apkbox /system/bin/apkbox.sh\n    class late_start\n    user root\n    group root\n    oneshot\n' | sudo tee sys/etc/init/apkbox.rc >/dev/null
sudo chmod 644 sys/etc/init/apkbox.rc
sudo dd if=/dev/zero of=sys/zero bs=1M || true; sudo rm -f sys/zero; sync
sudo umount sys

# ---- repack (kernel only supports gzip squashfs) ----
mkdir -p disk/android/data
mksquashfs sfs disk/android/system.sfs -comp gzip -b 128K -no-progress >/dev/null
cp iso/ramdisk.img disk/android/
truncate -s 3G base.img
mke2fs -q -t ext4 -L ANDROIDBOX -d disk base.img
qemu-img convert -c -O qcow2 base.img base.qcow2

# ---- smoke test: boot and screenshot ----
mkdir -p apk; printf 'PKG=com.android.settings\nW=480\nH=800\nD=240\n' > apk/app.conf; echo x > apk/app.apk
ACCEL=tcg; [ -e /dev/kvm ] && ACCEL=kvm
qemu-system-x86_64 -machine q35 -accel $ACCEL -cpu qemu64 -m 2048 -smp 2 -kernel iso/kernel -initrd iso/initrd.img \
  -append "root=/dev/ram0 androidboot.selinux=permissive SRC=/android DATA= console=ttyS0" \
  -drive file=base.qcow2,if=virtio,format=qcow2,snapshot=on -drive file=fat:rw:apk,format=raw,if=none,id=apk -device ide-hd,drive=apk,bus=ide.2 \
  -vga std -nic none -display none -serial unix:ser.sock,server,nowait -monitor unix:mon.sock,server,nowait &
QPID=$!
sleep 3
python3 "$ROOT/apkbox/image/console.py" ser.sock console.log &
CPID=$!
OK=0
for i in $(seq 1 36); do
  sleep 10
  python3 - <<'PY'
import socket,time
s=socket.socket(socket.AF_UNIX);s.connect('mon.sock');time.sleep(.3);s.recv(4096)
s.send(b'screendump shot.ppm\n');time.sleep(1.5)
PY
  convert shot.ppm screenshot.png
  M=$(convert screenshot.png -colorspace Gray -format "%[fx:mean]" info:)
  echo "t=$((i*10))s brightness=$M"
  # A bright boot logo is not proof of a completed Android boot.
  if grep -q "@@BOOT=1" console.log && awk "BEGIN{exit !($M>0.2)}"; then OK=1; sleep 20; python3 - <<'PY'
import socket,time
s=socket.socket(socket.AF_UNIX);s.connect('mon.sock');time.sleep(.3);s.recv(4096)
s.send(b'screendump shot.ppm\n');time.sleep(1.5)
PY
    convert shot.ppm screenshot.png; break; fi
done
kill $QPID || true; kill $CPID || true
tail -n 80 console.log || true
cp console.log "$ROOT/console.log" || true
[ $OK = 1 ] || { echo "SMOKE TEST FAILED"; cp screenshot.png "$ROOT/screenshot.png" || true; exit 1; }

# ---- portable QEMU for Windows ----
F=$(curl -s https://qemu.weilnetz.de/w64/ | grep -o 'qemu-w64-setup-[0-9]*\.exe' | sort -u | tail -1)
[ -n "$F" ] || { echo "::error::Unable to find Windows QEMU installer"; exit 1; }
download "https://qemu.weilnetz.de/w64/$F" qemu-setup.exe
./7zz x -y qemu-setup.exe -oqemu >/dev/null
QD=$(dirname "$(find qemu -name qemu-system-x86_64.exe | head -1)")
find "$QD" -maxdepth 1 -name 'qemu-system-*.exe' ! -name 'qemu-system-x86_64.exe' -delete
find "$QD" -maxdepth 1 -name '*.exe' ! -name 'qemu-system-x86_64.exe' ! -name 'qemu-img.exe' -delete
rm -rf "$QD/\$PLUGINSDIR" "$QD/doc"
cp iso/kernel iso/initrd.img base.qcow2 "$QD/"
touch "$QD/.keep"
(cd "$QD" && zip -qr "$ROOT/runtime.zip" .)
cd "$ROOT"; sha256sum runtime.zip | tee runtime.zip.sha256
cp "$W/screenshot.png" "$ROOT/screenshot.png"
ls -la runtime.zip
