# POCKETJS VERIFICATION — pocket-files

Round: PocketJS refactor. Device acceptance is a later pass.

## Shape

- Pocket app: `apps/pocket-files/pocket/`
- Frozen React reference: `apps/pocket-files/frontend/` (design reference; not extended).
- TSX files in the port: **6** (`main.tsx`, `app.tsx`, `ui.tsx`, three screens).
- Native tree only: `View`/`Text` from `@pocketjs/framework/solid/components`; no DOM, iframe, texture screenshot, or `dangerouslySetInnerHTML`.

## Manifest

- id: `dev.pocket-files.pocket`
- entry: `src/main.tsx`
- output: `pocket-files`
- viewport: `360×240 @2x` → physical `720×480`
- capabilities: `text.glyphs.baked`, `input.buttons`
- target: `rocknix-h700-kms`

## Admission / compile

```sh
cd apps/pocket-files/pocket
./setup-sdk.sh
SDK=${POCKETJS_SDK:-$HOME/development/.pocketjs-star-drift-sdk}
bun "$SDK/tools/pocket.ts" check \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json
bun "$SDK/tools/pocket.ts" compile \
  --runtime-config "$POCKET_VERIFY/pjs/profiles/pocket.runtime-config.ts" \
  --target rocknix-h700-kms --manifest pocket.json --output out.pocket
```

Admission: `sha256:99e91550236fb69f4f5e1a327010e3109e352122e156c84e7d435c7952255259`; TypeScript passes for the app module graph. Package: `out.pocket`, sha256 `b7fa4ffc36354abe46bcf38a0f0d04d341fa07e83ffb501b5b922189482d0290`.

## Frames

Rendered with the SDK wasm core at `renderScaled(2)` (720×480), not a browser:

```sh
cd "$POCKET_VERIFY"
bun tools/render-pocket.ts \
  --dir repos/pocket-files/apps/pocket-files/pocket \
  --script repos/pocket-files/apps/pocket-files/pocket/frames.json \
  --out repos/pocket-files/docs/pocket
```

| file | state | sha256 (first 16) |
|---|---|---|
| `docs/pocket/screen-home.png` | home | `48366b2957aab17a` |
| `docs/pocket/screen-home-focus.png` | home-focus | `22a9290c8342156b` |
| `docs/pocket/screen-detail.png` | detail | `e0e32b3186974a0b` |
| `docs/pocket/screen-detail-action.png` | detail-action | `ca4edc6b6e9e5406` |
| `docs/pocket/screen-settings.png` | settings | `51ad8e3472db0017` |
| `docs/pocket/screen-settings-toggled.png` | settings-toggled | `2064d0b87b2f0c3e` |

All six files have distinct bytes; the renderer rejects duplicate frames. The
tape is: move focus, open the item, run the first detail action, switch to
settings, move a row, toggle it.

Screens: Volumes/recent files, volume detail, browser settings. Interactions: CIRCLE opens; OPEN/COPY update state; DELETE refuses volumes and trashes files.

## Adapter

- **fixture** — this port runs from JSON baked into the package and labels the
  header `fixture`. `client.ts` also contains the `http` and `bridge` adapters
  behind the same `list()/get()/action()` surface.
- `http` uses `@pocketjs/framework/net-api`; valid only in the host dev profile.
- `bridge` uses `host.__pvc(JSON.stringify(request))`; the runtime forwards to
  the Go daemon on `127.0.0.1:<port>` with a hard timeout.
- Device gap: filesd bridge (files, transfers, trash/undo). No absolute host paths are exposed.
- Daemon contract: /api/storage, /api/files, /api/transfers; mock volumes.

## Not done / open questions

- Device run, saves/annotations and the bridge itself are out of this round.
- Real media/audio, OAuth, emulator, engine and system writes stay unimplemented
  where the original product already marked them mock or unavailable.
- Long lists use the six-item fixture surface; paging/virtual lists are a
  follow-up once the bridge provides paginated data.
