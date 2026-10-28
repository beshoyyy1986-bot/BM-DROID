#!/system/bin/sh
# find the virtual drive that carries app.apk / app.conf
mkdir -p /mnt/apk
while true; do
  for d in /dev/block/sd[a-z]1 /dev/block/sd[a-z]; do
    [ -b "$d" ] || continue
    mount -o ro $d /mnt/apk 2>/dev/null && [ -f /mnt/apk/app.apk ] && break 2
    umount /mnt/apk 2>/dev/null
  done
  sleep 1
done
until [ "$(getprop sys.boot_completed)" = 1 ]; do sleep 1; done
. /mnt/apk/app.conf
settings put global device_provisioned 1
settings put secure user_setup_complete 1
settings put global policy_control immersive.full=*
settings put global stay_on_while_plugged_in 7
settings put system screen_off_timeout 2147483647
wm size ${W}x${H}; wm density $D
CUR=$(cat /data/apkbox.sha 2>/dev/null)
NEW=$(md5sum /mnt/apk/app.apk | cut -d' ' -f1)
if [ "$CUR" != "$NEW" ] || ! pm path $PKG >/dev/null 2>&1; then
  cp /mnt/apk/app.apk /data/local/tmp/app.apk
  pm install -r /data/local/tmp/app.apk && echo $NEW > /data/apkbox.sha
  rm -f /data/local/tmp/app.apk
fi
while true; do
  monkey -p $PKG -c android.intent.category.LAUNCHER 1 >/dev/null 2>&1
  sleep 8
  while pidof $PKG >/dev/null 2>&1; do sleep 3; done
done
