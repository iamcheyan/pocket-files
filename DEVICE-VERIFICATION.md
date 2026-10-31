# DEVICE VERIFICATION — pocket-files

Status: **DEVICE-PASS** for the captured states. Local host verification is in
`apps/pocket-files/POCKETJS-VERIFICATION.md`.

## Device

- Anbernic RG34XX, Base OS 1.5.2 (h700), kernel 4.9.170 aarch64.
- Access: Debian host -> HX90 -> USB ADB.
- Display: `/dev/fb0`, 720x480 visible page, 32 bpp.
- Runtime: `/mnt/sdcard/System/pocketjs/bin/pocket-linux-kms` (existing boot runtime, not rebuilt or replaced).
- Package: `pocket-files` sha256 `1f77d4ad42c973f36e39c9120f134e22814dd5a9afa09a93fde623b3afc2d2b5`.
- Deployed to `/storage/pjs/pocket-files/app.pocket` only.

## Procedure

1. Recorded boot launcher pid `17713` and `boot.pocket`
   sha256 `6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`.
2. Paused the launcher supervisor, stopped the launcher, and started the
   package with the same runtime and `/dev/fb0`.
3. Created a virtual keyboard with `/dev/uinput` and drove the native view
   tree through a FIFO (`uinputd-c`); no host browser and no HTML.
4. Captured `/dev/fb0` with `dd` -> base64 -> PIL.
5. Killed the package, restored the launcher, and read the pid
   `18419` and boot hash back.

Result: launcher pid changed, boot package sha256 stayed
`6ed41612bce1723d1eda64b5a0ee9333a105ed498b7085c03267539cf6c1b32d`. No write to `/storage/System`, `/storage/pocketjs`,
`/storage/baseos.conf` or `/storage/content`.

## Frames

The images are the device framebuffer. The UI was rasterized by the PocketJS
native renderer on the RG34XX, not by host Chrome.

| file | sha256 (first 16) |
|---|---|
| `docs/device/device-files-detail-action.png` | `b67556c3cc195499` |
| `docs/device/device-files-detail.png` | `74d28307d7c1396b` |
| `docs/device/device-files-fb-after.png` | `d59130e3329b8496` |
| `docs/device/device-files-fb-before.png` | `d59130e3329b8496` |
| `docs/device/device-files-home-focus.png` | `812404745adbde46` |
| `docs/device/device-files-home.png` | `26e7f0315105fe06` |
| `docs/device/device-files-settings-toggled.png` | `b008c71cba92b636` |
| `docs/device/device-files-settings.png` | `6bb8acbc6c8a794a` |

`device-files-fb-before.png` and `device-files-fb-after.png` are the boot
launcher before and after; they are byte-identical here, which is the intended
restore evidence.

Captured states: 6 states: home, focus, detail, detail action, settings, settings toggle.
