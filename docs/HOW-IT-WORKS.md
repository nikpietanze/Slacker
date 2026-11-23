# How it works

## Why a texture?

WoW addons run in a sandbox. They can't read files, open sockets, see other processes or read the clipboard. Saved variables are read only at login or `/reload`. No supported channel exists for an outside program to push data into a running addon.

An addon *can* load a texture from its own folder. Once loaded, `Texture:GetSize()` on a texture that has no explicit size reports the image's **native pixel size**. That size is the bridge.

## Getting past the texture cache

The client caches each texture it loads. It keeps a texture while anything uses it, and frees unused ones in a cleanup that runs at irregular times: on a real WoW Forever 1.60.1 session, anywhere from 7 to 76 seconds apart. Loading the same file again before a cleanup returns the cached copy, even if the file changed on disk. Loading a file that hasn't been loaded since the last cleanup always reads it fresh.

So the bridge isn't one file but a **ring of 128 slots**, all holding the same value. The addon reads the next slot each second, and holds it only for the frame it takes to measure it. Each slot then rests unused for about two minutes before its next read, by which time a cleanup has almost always freed it. Changes reach the badge in about two seconds instead of waiting for the next cleanup.

This was worked out in game: with a single file read over and over, the badge got stuck on its first change.

## The bridge files

Each slot is two images in `Interface/AddOns/Slacker/Bridge/`: `sNNNt.tga` holds the tens digit and `sNNNo.tga` the ones digit, for `NNN` from `001` to `128`.

| Tens | Ones | Meaning |
|---|---|---|
| 0–9 | 0–9 | count 0–99 |
| marker | 1 | 100 or more, shown as `99+` |
| marker | marker | companion offline (the placeholders shipped with the addon) |

Each digit 0–10 (10 is the marker) maps to a pair of exponents `(a, b)`, listed diagonal by diagonal (`a+b = 0, 1, 2, …`), and the image is `2^(a+1) × 2^(b+1)` pixels. The largest is 64 pixels, 274 bytes as a 32-bit TGA, so a full update of all 128 slots is under 70 KB. Dimensions are powers of two, as WoW requires, and the images are 32-bit because an 8-bit grayscale TGA didn't load properly on WoW Forever.

The table is built identically in [`codec.go`](../companion/internal/bridge/codec.go) and [`Bridge.lua`](../Bridge.lua). [`addon_test.go`](../companion/internal/addontest/addon_test.go) loads the real Lua and checks every value decodes the same way in both. It also replays the client's caching behavior to check every change shows up within a few seconds.

Each image is written atomically (temp file, then rename), so WoW never reads a partial file. The companion writes the ones digit before the tens digit, and the addon only accepts a value once two reads in a row agree, so a slot caught mid-update is skipped. The companion rewrites the slots only when the count changes, and writes "offline" when it exits cleanly.

## The addon's read loop

Once a second ([`Bridge.lua`](../Bridge.lua)), the addon:

1. Points two hidden probe textures at the next slot's tens and ones images.
2. Waits a frame or two for them to load, reads `GetSize()` and releases them right away.
3. Divides by the measured size of its own 64×64 icon (a safeguard against UI scaling), rounds each to the nearest power of two, and decodes the two digits.
4. Applies the value once two reads in a row agree.

The work is two tiny texture loads per second, nothing running between reads, and no combat-restricted API.

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
