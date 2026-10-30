package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The fixture mirrors real mountinfo field layout, including a non-removable
// disk, a production data mount, a virtual filesystem and an unlabelled card.
const mountinfoFixture = `21 1 8:1 / / rw,relatime shared:1 - ext4 /dev/nvme0n1p2 rw
33 21 179:1 / /run/media/usb-stick rw,nosuid,nodev shared:33 - vfat /dev/mmcblk0p1 rw
44 21 8:17 / /run/media/card-a rw,nosuid,nodev shared:44 - exfat /dev/sdb1 rw
55 21 0:45 / /proc rw,nosuid shared:55 - proc proc rw
66 21 8:33 / /storage rw,relatime shared:66 - xfs /dev/sdc1 rw
77 21 179:33 / /run/media/ro-card ro,nosuid,nodev shared:77 - exfat /dev/mmcblk1p1 ro
88 21 8:49 / /run/media/unlabelled rw,nosuid,nodev shared:88 - vfat /dev/sdd1 rw
99 21 8:60 / /run/media/label\040with\040space rw,nosuid,nodev shared:99 - vfat /dev/sde1 rw
`

func fixtureEnv(t *testing.T) discoveryEnv {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("proc/self/mountinfo", mountinfoFixture)
	write("sys/class/block/mmcblk0/removable", "1\n")
	write("sys/class/block/sdb/removable", "1\n")
	write("sys/class/block/sdc/removable", "0\n")
	write("sys/class/block/mmcblk1/removable", "1\n")
	write("sys/class/block/sdd/removable", "1\n")
	write("sys/class/block/sde/removable", "1\n")
	if err := os.MkdirAll(filepath.Join(root, "dev/disk/by-uuid"), 0o755); err != nil {
		t.Fatal(err)
	}
	links := map[string]string{
		"1111-2222": "../../mmcblk0p1",
		"3333-4444": "../../sdb1",
		"7777-8888": "../../mmcblk1p1",
		"9999-0000": "../../sde1",
	}
	for uuid, target := range links {
		if err := os.Symlink(target, filepath.Join(root, "dev/disk/by-uuid", uuid)); err != nil {
			t.Fatal(err)
		}
	}
	return discoveryEnv{
		mountinfoPath: "/proc/self/mountinfo",
		readFile: func(path string) (string, error) {
			b, err := os.ReadFile(filepath.Join(root, path))
			return string(b), err
		},
		readLink: func(path string) (string, error) { return os.Readlink(filepath.Join(root, path)) },
		readDir: func(path string) ([]os.DirEntry, error) {
			return os.ReadDir(filepath.Join(root, path))
		},
		statfs: func(string) (int64, int64, error) { return 64 << 30, 16 << 30, nil },
	}
}

func TestDiscoveryReportsOnlyLabelledRemovableVolumes(t *testing.T) {
	devices := discoverRemovableStorage(fixtureEnv(t))
	ids := make([]string, 0, len(devices))
	for _, d := range devices {
		ids = append(ids, d.ID)
	}
	sort.Strings(ids)
	want := []string{"1111-2222", "3333-4444", "7777-8888", "9999-0000"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("removable ids = %v, want %v", ids, want)
	}
	for _, d := range devices {
		if strings.Contains(d.ID, "/dev/") || strings.Contains(d.Name, "/dev/") || strings.Contains(d.Name, "mmcblk") || strings.Contains(d.Name, "sd") {
			t.Errorf("kernel device name leaked into API-facing fields: %+v", d)
		}
		if d.Total != 64<<30 || d.Free != 16<<30 {
			t.Errorf("capacity not reported: %+v", d)
		}
	}
}

func TestDiscoveryMarksReadOnlyAndDecodesEscapes(t *testing.T) {
	devices := discoverRemovableStorage(fixtureEnv(t))
	byID := map[string]RemovableDevice{}
	for _, d := range devices {
		byID[d.ID] = d
	}
	if !byID["7777-8888"].ReadOnly {
		t.Errorf("ro mount not reported read-only: %+v", byID["7777-8888"])
	}
	if got := byID["9999-0000"].MountPoint; got != "/run/media/label with space" {
		t.Errorf("mountinfo escaping not decoded: %q", got)
	}
	if byID["3333-4444"].Kind != "usb" {
		t.Errorf("USB kind = %q", byID["3333-4444"].Kind)
	}
	if byID["1111-2222"].Kind != "tf" {
		t.Errorf("mmc kind = %q", byID["1111-2222"].Kind)
	}
	if !byID["1111-2222"].Removable {
		t.Errorf("removable flag lost")
	}
}

func TestDiscoverySkipsUnlabelledAndProtectsProductionMount(t *testing.T) {
	devices := discoverRemovableStorage(fixtureEnv(t))
	for _, d := range devices {
		if strings.Contains(d.MountPoint, "/storage") {
			t.Fatalf("production data mount was reported as removable: %+v", d)
		}
		if d.ID == "" {
			t.Fatalf("device without a stable UUID was reported: %+v", d)
		}
	}
}

func TestDiscoveryHandlesMissingSources(t *testing.T) {
	empty := discoveryEnv{
		mountinfoPath: "/proc/self/mountinfo",
		readFile:      func(string) (string, error) { return "", os.ErrNotExist },
		readLink:      os.Readlink,
		readDir:       os.ReadDir,
		statfs:        nil,
	}
	if got := discoverRemovableStorage(empty); got != nil {
		t.Fatalf("expected no devices without mountinfo, got %+v", got)
	}
	noSysfs := fixtureEnv(t)
	noSysfs.readFile = func(path string) (string, error) {
		if path == noSysfs.mountinfoPath {
			return mountinfoFixture, nil
		}
		return "", os.ErrNotExist
	}
	if got := discoverRemovableStorage(noSysfs); len(got) != 0 {
		t.Fatalf("expected conservative empty result without sysfs, got %+v", got)
	}
}
