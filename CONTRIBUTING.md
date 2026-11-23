# Contributing

Thanks for helping out. A few ground rules keep Slacker small and safe to run:

- **Keep the scope.** The addon answers "do I need to check Slack?" and nothing more. Message previews, senders, channel names and Slack API access are out of scope (see "Non-goals" in [docs/HANDOFF.md](docs/HANDOFF.md)).
- **Privacy first.** The companion makes no network connections and never reads notification text. Windows and Linux look only at which app a notification came from, and macOS reads only a number.
- **No taint.** Plain frames only: no secure templates or protected calls. Test in and out of combat with `/console taintLog 1`.
- **Test in game.** Say in your PR what you tested, on which client build and OS, and what you couldn't verify.
- **No copied code** from addons with incompatible licenses.
- Use tabs for indentation in Lua.

## Layout

| Path | What |
|---|---|
| `Slacker.toc`, `*.lua` | The addon, at the repo root so you can clone it straight into `AddOns`. `Core.lua` holds settings and commands, `Bridge.lua` reads the companion, `Icon.lua` draws the icon, and `Options.lua` builds the options panel. |
| `Libs/` | LibStub and LibEditMode, bundled unmodified |
| `Media/`, `Bridge/` | Textures (generated, see below) and the bridge placeholder |
| `companion/` | The companion app in Go. One static binary per OS. |
| `companion/internal/source/` | One notification watcher per OS |
| `companion/internal/bridge/` | Count-to-texture-size codec, file writing, and addon folder discovery |
| `companion/internal/addontest/` | Runs the addon's Lua against a mocked client |
| `companion/cmd/genart/` | Draws `Media/*.tga`, `docs/logo/logo.png`, the tray icons in `companion/assets/` and the Windows `.exe` icon, all from code |
| `scripts/build-macos.sh` | Builds and zips `Slacker Companion.app` |

`.pkgmeta` keeps `companion/`, `docs/` and the repo files out of the addon zip.

## Building and testing

You need Go (version in `companion/go.mod`).

```sh
make test    # companion tests + the addon's Lua against a mocked client
make build   # Windows and Linux companions into dist/
make macos   # macOS app (on a Mac: the menu bar icon needs cgo)
make art     # redraw the addon textures, logo, tray icons and app icons
```

`make art` draws the default `round` style. To try the others, run `go run ./cmd/genart -style flat ..` (or `glass`) from `companion/`.

To try it in game, symlink the repo into `_forever_/Interface/AddOns/Slacker`. Then run `go run . demo --wow-dir "<WoW folder>"` from `companion/` and turn on `/slacker debug`. A bare `go run .` with no arguments behaves like a double-click: it installs itself and turns on start at login. Use `go run . run` while developing. The companion rewrites the 256 files in `Bridge/` while it runs. Don't commit them unless they're the "offline" placeholders (`go run . set off` puts them back), or hide the churn with `git update-index --skip-worktree Bridge/*.tga`.

Before a release, go through [docs/TESTING.md](docs/TESTING.md).

## Releases

Add a CHANGELOG entry and push a `vX.Y.Z` tag. The BigWigs packager builds the addon zip (and uploads it to CurseForge once `CF_API_KEY` and `X-Curse-Project-ID` are set). CI then attaches the companion binaries to the same release.

## Adding a notification source

Implement `source.Source` (`Run`, `Diagnose`, `Name`) in a new build-tagged file under `companion/internal/source/`. `Run` calls `report(count)` whenever the count may have changed. Duplicate reports are fine.
