package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const magic = "APKBOX01"

type Meta struct {
	AppName       string `json:"app_name"`
	AppPackage    string `json:"app_package"`
	LicenseKey    string `json:"license_key"`
	APIBase       string `json:"api_base"`
	RuntimeURL    string `json:"runtime_url"`
	RuntimeSHA256 string `json:"runtime_sha256"`
}

var self string

func msg(title, text string, icon int) {
	// icon: 0x10 error, 0x40 info
	u := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	u.Call(0, uintptr(unsafePtr(t)), uintptr(unsafePtr(c)), uintptr(icon))
}

func fail(meta *Meta, text string) {
	name := "APK Box"
	if meta != nil {
		name = meta.AppName
	}
	msg(name, text, 0x10)
	os.Exit(1)
}

// readPayload reads the metadata and APK appended to this executable.
func readPayload() (*Meta, []byte, error) {
	f, err := os.Open(self)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	st, _ := f.Stat()
	foot := make([]byte, 24)
	if _, err := f.ReadAt(foot, st.Size()-24); err != nil {
		return nil, nil, err
	}
	if string(foot[16:]) != magic {
		return nil, nil, fmt.Errorf("no payload")
	}
	metaLen := int64(binary.LittleEndian.Uint64(foot[0:8]))
	apkLen := int64(binary.LittleEndian.Uint64(foot[8:16]))
	start := st.Size() - 24 - metaLen - apkLen
	buf := make([]byte, metaLen+apkLen)
	if _, err := f.ReadAt(buf, start); err != nil {
		return nil, nil, err
	}
	var m Meta
	if err := json.Unmarshal(buf[apkLen:], &m); err != nil {
		return nil, nil, err
	}
	return &m, buf[:apkLen], nil
}

func ps(script string) (string, error) {
	c := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := c.Output()
	return strings.TrimSpace(string(out)), err
}

func hwid() string {
	out, _ := ps(`(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Cryptography').MachineGuid`)
	h := sha256.Sum256([]byte("apkbox|" + out))
	return hex.EncodeToString(h[:])
}

type apiResp struct {
	OK           bool   `json:"ok"`
	Error        string `json:"error"`
	SelfDestruct bool   `json:"self_destruct"`
	SessionToken string `json:"session_token"`
	CheckAfter   int    `json:"check_after_seconds"`
	Config       struct {
		Cores   int `json:"cores"`
		RAM     int `json:"ram"`
		Width   int `json:"width"`
		Height  int `json:"height"`
		Density int `json:"density"`
	} `json:"config"`
}

func call(m *Meta, op string, body map[string]string) (*apiResp, error) {
	b, _ := json.Marshal(body)
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Post(strings.TrimRight(m.APIBase, "/")+"/"+op, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r apiResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// license: first run activates (binds device + creates session), later runs validate with the saved session.
func license(m *Meta, id, base string) (*apiResp, error) {
	sf := filepath.Join(base, "session.txt")
	if tok, err := os.ReadFile(sf); err == nil {
		return call(m, "validate", map[string]string{"session_token": strings.TrimSpace(string(tok)), "device_hash": id, "app_version": "1"})
	}
	r, err := call(m, "activate", map[string]string{"license_key": m.LicenseKey, "device_hash": id, "app_version": "1"})
	if err == nil && r.OK {
		os.MkdirAll(base, 0755)
		os.WriteFile(sf, []byte(r.SessionToken), 0600)
	}
	return r, err
}

var reasons = map[string]string{
	"invalid_credentials": "الترخيص غير صالح أو مرتبط بجهاز آخر.",
	"already_bound":       "هذا الترخيص مرتبط بجهاز آخر.",
	"banned":              "تم إيقاف الترخيص.",
	"paused":              "الاشتراك متوقف مؤقتًا.",
	"expired":             "انتهى الاشتراك.",
}

func denied(m *Meta, base string, r *apiResp) {
	if r.SelfDestruct {
		os.RemoveAll(base)
	}
	t := reasons[r.Error]
	if t == "" {
		t = "تعذّر التحقق من الترخيص (" + r.Error + ")."
	}
	fail(m, t)
}

func requirements() error {
	out, err := ps(`(Get-CimInstance Win32_ComputerSystem).HypervisorPresent`)
	if err == nil && strings.EqualFold(out, "True") {
		return nil
	}
	// try to enable Windows Hypervisor Platform (needs admin)
	if _, err := ps(`Enable-WindowsOptionalFeature -Online -FeatureName HypervisorPlatform -NoRestart`); err != nil {
		return fmt.Errorf("يلزم تفعيل Windows Hypervisor Platform وVirtualization من الـBIOS.\nشغّل البرنامج كمسؤول مرة واحدة ثم أعد تشغيل الجهاز.")
	}
	return fmt.Errorf("تم تفعيل Windows Hypervisor Platform. أعد تشغيل الجهاز ثم افتح البرنامج.")
}

func download(url, dst, wantSHA string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()
	if wantSHA != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), wantSHA) {
		return fmt.Errorf("checksum mismatch")
	}
	return nil
}

func unzip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		p := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(p, filepath.Clean(dst)+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(p, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(p), 0755)
		rc, _ := f.Open()
		w, _ := os.Create(p)
		io.Copy(w, rc)
		w.Close()
		rc.Close()
	}
	return nil
}

func shortcut(m *Meta, target string) {
	ps(fmt.Sprintf(`$s=(New-Object -ComObject WScript.Shell).CreateShortcut([Environment]::GetFolderPath('Desktop')+'\%s.lnk');$s.TargetPath='%s';$s.WorkingDirectory='%s';$s.Save()`,
		m.AppName, target, filepath.Dir(target)))
	ps(fmt.Sprintf(`$s=(New-Object -ComObject WScript.Shell).CreateShortcut([Environment]::GetFolderPath('Programs')+'\%s.lnk');$s.TargetPath='%s';$s.WorkingDirectory='%s';$s.Save()`,
		m.AppName, target, filepath.Dir(target)))
}

func main() {
	self, _ = os.Executable()
	m, apk, err := readPayload()
	if err != nil {
		fail(nil, "ملف غير صالح.")
	}
	id := hwid()

	// 1) license (online)
	base := filepath.Join(os.Getenv("LOCALAPPDATA"), "ApkBox", m.AppName)
	r, err := license(m, id, base)
	if err != nil {
		fail(m, "تعذّر الاتصال بالإنترنت للتحقق من الترخيص.")
	}
	if !r.OK {
		denied(m, base, r)
	}

	os.MkdirAll(filepath.Join(base, "apk"), 0755)
	installed := filepath.Join(base, m.AppName+".exe")
	ready := filepath.Join(base, "runtime", ".ready")

	// 2) setup: requirements + runtime download (first run only)
	if _, err := os.Stat(ready); err != nil {
		if err := requirements(); err != nil {
			fail(m, err.Error())
		}
		msg(m.AppName, "سيتم الآن تنزيل المتطلبات (مرة واحدة فقط). قد يستغرق ذلك عدة دقائق. اضغط موافق للبدء.", 0x40)
		zp := filepath.Join(base, "runtime.zip")
		if err := download(m.RuntimeURL, zp, m.RuntimeSHA256); err != nil {
			fail(m, "فشل تنزيل المتطلبات: "+err.Error())
		}
		if err := unzip(zp, filepath.Join(base, "runtime")); err != nil {
			fail(m, "فشل فك الضغط.")
		}
		os.Remove(zp)
		os.WriteFile(ready, []byte("ok"), 0644)
	}
	if self != installed {
		if in, err := os.ReadFile(self); err == nil {
			os.WriteFile(installed, in, 0755)
		}
	}
	shortcut(m, installed)

	// 3) APK goes to a virtual FAT drive, installed by the guest on boot
	os.WriteFile(filepath.Join(base, "apk", "app.apk"), apk, 0644)
	cfg := r.Config
	if cfg.RAM == 0 {
		cfg.RAM, cfg.Cores, cfg.Width, cfg.Height, cfg.Density = 1024, 1, 480, 800, 240
	}
	os.WriteFile(filepath.Join(base, "apk", "app.conf"), []byte(fmt.Sprintf("PKG=%s\nW=%d\nH=%d\nD=%d\n", m.AppPackage, cfg.Width, cfg.Height, cfg.Density)), 0644)
	rt := filepath.Join(base, "runtime")
	userDisk := filepath.Join(base, "user.qcow2")
	if _, err := os.Stat(userDisk); err != nil {
		c := exec.Command(filepath.Join(rt, "qemu-img.exe"), "create", "-f", "qcow2", "-b", filepath.Join(rt, "base.qcow2"), "-F", "qcow2", userDisk)
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		c.Run()
	}
	if cfg.RAM < 1536 {
		cfg.RAM = 1536
	}
	if cfg.Cores < 2 {
		cfg.Cores = 2
	}
	q := exec.Command(filepath.Join(rt, "qemu-system-x86_64.exe"),
		"-machine", "q35", "-accel", "whpx,kernel-irqchip=off", "-cpu", "qemu64",
		"-m", fmt.Sprint(cfg.RAM), "-smp", fmt.Sprint(cfg.Cores),
		"-kernel", filepath.Join(rt, "kernel"), "-initrd", filepath.Join(rt, "initrd.img"),
		"-append", "root=/dev/ram0 androidboot.selinux=permissive SRC=/android DATA= quiet",
		"-drive", "file="+userDisk+",if=virtio,format=qcow2",
		"-drive", "file=fat:rw:"+filepath.Join(base, "apk")+",format=raw,if=none,id=apk",
		"-device", "ide-hd,drive=apk,bus=ide.2",
		"-device", "virtio-vga", "-audiodev", "dsound,id=snd", "-device", "intel-hda", "-device", "hda-duplex,audiodev=snd",
		"-nic", "user,model=virtio-net-pci",
		"-device", "qemu-xhci", "-device", "usb-tablet", "-display", "sdl,gl=off", "-name", m.AppName)
	q.Dir = rt
	q.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := q.Start(); err != nil {
		fail(m, "فشل التشغيل: "+err.Error())
	}

	// 4) keep checking license online while running
	done := make(chan struct{})
	go func() { q.Wait(); close(done) }()
	t := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-done:
			return
		case <-t.C:
			r, err := license(m, id, base)
			if err == nil && !r.OK {
				q.Process.Kill()
				denied(m, base, r)
			}
		}
	}
}
