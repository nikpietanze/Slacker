# How it works

## Why a texture?

WoW addons run in a sandbox. They can't read files, open sockets, see other processes or read the clipboard. Saved variables are read only at login or `/reload`. No supported channel exists for an outside program to push data into a running addon.

An addon *can* load a texture from its own folder. Once loaded, `Texture:GetSize()` on a texture that has no explicit size reports the image's **native pixel size**. That size is the bridge.

## The bridge file

The companion writes `Interface/AddOns/SlackNotify/Bridge/count.tga`: a blank 8-bit grayscale TGA whose width and height encode one value.

| Value | Meaning |
|---|---|
| code 0 | companion offline (also the 2×2 placeholder shipped with the addon) |
| code 1–100 | count 0–99 |
| code 101 | 100 or more, shown as `99+` |

Each code maps to a pair of exponents `(a, b)`, and the file is `2^(a+1) × 2^(b+1)` pixels. Pairs are listed diagonal by diagonal (`a+b = 0, 1, 2, …`, each at most 10) so the common small counts give tiny files. Zero notifications is a 2×4 file of 26 bytes, and anything up to 10 stays under 300 bytes. The largest files, for counts in the 90s and `99+`, are about 128 KB, which is still only a cached disk read. Dimensions are always powers of two, as WoW requires.

The table is built identically in [`codec.go`](../companion/internal/bridge/codec.go) and [`Bridge.lua`](../Bridge.lua). [`addon_test.go`](../companion/internal/addontest/addon_test.go) loads the real Lua and checks every code decodes the same way in both.

Writes are atomic: write `count.tga.tmp`, then rename it over `count.tga`. WoW never sees a partial file. The companion rewrites the file only when the count changes, and writes "offline" when it exits cleanly.

## The addon's read loop

About once a second ([`Bridge.lua`](../Bridge.lua)), the addon:

1. Clears its hidden probe texture (`SetTexture(nil)`), so the client can drop its cached copy.
2. 0.15 s later, sets the texture to the bridge file again.
3. Waits for `IsObjectLoaded()`, reads `GetSize()`, and decodes the count.

A size that isn't a valid bridge value is ignored, so a half-loaded texture never flickers the badge. The work is one tiny texture load per second, no `OnUpdate` running between reads, and no combat-restricted API.

## Where the count comes from

### Windows

Windows keeps the notification center's toasts in a per-user SQLite database, `%LOCALAPPDATA%\Microsoft\Windows\Notifications\wpndatabase.db`. The companion opens it **read-only** through `winsqlite3.dll`, which ships with Windows, so there's no bundled SQLite and no cgo. It queries only when the database files change, and counts toasts whose handler is Slack (`com.squirrel.slack.slack`, or the Microsoft Store package).

Clearing: when the Slack window is in the foreground, every Slack toast seen so far counts as read. Toasts you dismiss, or that Slack removes, leave the database and drop out of the count.

### macOS

Slack keeps a badge on its Dock icon with its own count of notification-worthy unreads. `lsappinfo info -only StatusLabel Slack` reads that badge without needing any permission, every 2 s. A bare `•` (unread but not notifying) counts as 0.

Reading the Notification Center database directly would need Full Disk Access, which is a bigger ask for less accurate data, since the badge already clears as you read.

### Linux

The companion becomes a passive D-Bus monitor on your own session bus, which needs no root:

- `org.freedesktop.Notifications.Notify` calls with an app name containing "slack" add a notification. The id is taken from the server's reply.
- `NotificationClosed` with reason 2 (dismissed) or 3 (closed by Slack) removes it. Reason 1 (popup expired into the tray) keeps it counted.
- If Slack publishes a `com.canonical.Unity.LauncherEntry` badge, that count is Slack's own and takes precedence.

## What is never done

- No Slack API, token, OAuth, bot or workspace app.
- No network connections of any kind.
- No reading of notification text: Windows and Linux look only at which app a notification belongs to, and macOS reads only a number.
- No game memory reading, input injection or screen capture.
