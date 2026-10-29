# Talks to the Android shell on the serial console, to collect boot diagnostics.
import socket, sys, time, threading
sock, out = sys.argv[1], sys.argv[2]
s = socket.socket(socket.AF_UNIX); s.connect(sock)
f = open(out, 'wb', buffering=0)
def rd():
    while True:
        try:
            d = s.recv(4096)
            if not d: break
            f.write(d)
        except Exception: break
threading.Thread(target=rd, daemon=True).start()
CMDS = ['echo @@BOOT=$(getprop sys.boot_completed) ANIM=$(getprop init.svc.bootanim)',
        'ps -A | grep -E "system_server|zygote|surfaceflinger|apkbox"',
        'logcat -d -b all | grep -E "FATAL EXCEPTION|Watchdog|has died|SystemServer|ANR in|Exception|Slow operation" | tail -60',
        'logcat -d -b all | tail -40',
        'echo @@PACKAGE=$(pm path com.neumorphic.calculator)',
        'echo @@APP_PID=$(pidof com.neumorphic.calculator)',
        'dumpsys activity activities | grep mResumedActivity']
for t in (90, 200, 300):
    time.sleep(t - (time.time() % 1000) if False else 0)
start = time.time()
for t in (120, 240):
    while time.time() - start < t: time.sleep(1)
    for c in CMDS:
        s.send((c + '\n').encode()); time.sleep(4)
time.sleep(600)
