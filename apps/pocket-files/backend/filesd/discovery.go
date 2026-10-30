package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RemovableDevice is a read-only description of an attached Linux block device.
// Kernel and Node are internal correlation aids: they are never serialized by
// the API, which exposes only ID (a stable filesystem UUID) and a friendly name.
type RemovableDevice struct {
	ID         string
	Name       string
	Kind       string
	MountPoint string
	ReadOnly   bool
	Removable  bool
	Filesystem string
	Total      int64
	Free       int64
	Kernel     string
	Node       string
}

type discoveryEnv struct {
	mountinfoPath string
	readFile      func(string) (string, error)
	readLink      func(string) (string, error)
	readDir       func(string) ([]os.DirEntry, error)
	statfs        func(string) (total, free int64, err error)
}

func defaultDiscoveryEnv() discoveryEnv {
	return discoveryEnv{
		mountinfoPath: "/proc/self/mountinfo",
		readFile:      func(path string) (string, error) { b, err := os.ReadFile(path); return string(b), err },
		readLink:      os.Readlink,
		readDir:       os.ReadDir,
		statfs:        statfsUsage,
	}
}

// discoverRemovableStorage reports attached removable filesystems without
// mounting, unmounting or writing anything. Identification is conservative:
// anything not positively marked removable in sysfs is omitted.
func discoverRemovableStorage(env discoveryEnv) []RemovableDevice {
	content, err := env.readFile(env.mountinfoPath)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var devices []RemovableDevice
	for _, line := range strings.Split(content, "\n") {
		dev, ok := parseMountinfoLine(line, env)
		if !ok || seen[dev.ID] {
			continue
		}
		seen[dev.ID] = true
		devices = append(devices, dev)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].MountPoint < devices[j].MountPoint })
	return devices
}

// parseMountinfoLine parses one /proc/self/mountinfo record:
// id parent major:minor root mountpoint options - fstype source superopts
func parseMountinfoLine(line string, env discoveryEnv) (RemovableDevice, bool) {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return RemovableDevice{}, false
	}
	separator := -1
	for i, f := range fields {
		if f == "-" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+2 >= len(fields) {
		return RemovableDevice{}, false
	}
	node := unescapeMountinfo(fields[separator+2])
	mountPoint := unescapeMountinfo(fields[4])
	fsType := fields[separator+1]
	if !strings.HasPrefix(node, "/dev/") || isVirtualFilesystem(fsType) || isExcludedMountPoint(mountPoint) {
		return RemovableDevice{}, false
	}
	kernel := filepath.Base(node)
	if !sysfsRemovable(kernel, env) {
		return RemovableDevice{}, false
	}
	id := stableDeviceID(node, env)
	if id == "" {
		return RemovableDevice{}, false
	}
	kind := "usb"
	if strings.HasPrefix(kernel, "mmcblk") {
		kind = "tf"
	}
	dev := RemovableDevice{
		ID: id, Name: kindDisplayName(kind), Kind: kind, MountPoint: mountPoint,
		ReadOnly: hasMountOption(fields[5], "ro"), Removable: true, Filesystem: fsType,
		Kernel: kernel, Node: node,
	}
	if env.statfs != nil {
		if total, free, err := env.statfs(mountPoint); err == nil {
			dev.Total, dev.Free = total, free
		}
	}
	return dev, true
}

func sysfsRemovable(kernel string, env discoveryEnv) bool {
	for _, candidate := range []string{kernel, stripPartition(kernel)} {
		if content, err := env.readFile(filepath.Join("/sys/class/block", candidate, "removable")); err == nil {
			return strings.TrimSpace(content) == "1"
		}
	}
	return false
}

func unescapeMountinfo(value string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(value)
}

func hasMountOption(options, want string) bool {
	for _, option := range strings.Split(options, ",") {
		if option == want {
			return true
		}
	}
	return false
}

func isVirtualFilesystem(fsType string) bool {
	switch fsType {
	case "proc", "sysfs", "devtmpfs", "devpts", "tmpfs", "cgroup", "cgroup2", "overlay",
		"efivarfs", "debugfs", "tracefs", "securityfs", "pstore", "bpf", "configfs",
		"fusectl", "mqueue", "hugetlbfs", "autofs", "binfmt_misc", "nsfs", "ramfs",
		"squashfs", "nfs", "nfs4", "cifs", "smb3", "fuse.sshfs", "zfs":
		return true
	}
	return false
}

// Production data mounts on this host are never presented or probed as
// removable storage, even if a fixture or unusual kernel reports them so.
func isExcludedMountPoint(mountPoint string) bool {
	switch mountPoint {
	case "/", "/nix", "/nix/store", "/boot", "/home":
		return true
	}
	for _, prefix := range []string{"/nix/", "/boot/", "/storage"} {
		if strings.HasPrefix(mountPoint, prefix) {
			return true
		}
	}
	return false
}

func stripPartition(kernel string) string {
	for _, prefix := range []string{"mmcblk", "nvme", "loop", "md"} {
		if !strings.HasPrefix(kernel, prefix) {
			continue
		}
		if idx := strings.LastIndex(kernel, "p"); idx > len(prefix)-1 {
			if _, err := strconv.Atoi(kernel[idx+1:]); err == nil {
				return kernel[:idx]
			}
		}
		return kernel
	}
	if trimmed := strings.TrimRight(kernel, "0123456789"); trimmed != "" {
		return trimmed
	}
	return kernel
}

// stableDeviceID resolves a partitions's filesystem UUID through
// /dev/disk/by-uuid. Without a UUID the device is skipped rather than being
// given a guessed slot name such as "TF-A".
func stableDeviceID(node string, env discoveryEnv) string {
	entries, err := env.readDir("/dev/disk/by-uuid")
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		target, linkErr := env.readLink(filepath.Join("/dev/disk/by-uuid", entry.Name()))
		if linkErr != nil {
			continue
		}
		if filepath.Base(target) == filepath.Base(node) {
			return strings.ToLower(entry.Name())
		}
	}
	return ""
}

func kindDisplayName(kind string) string {
	if kind == "tf" {
		return "Memory card"
	}
	return "USB storage"
}

func statfsUsage(path string) (total, free int64, err error) {
	total, _, free, err = statfsNumbers(path)
	return total, free, err
}
