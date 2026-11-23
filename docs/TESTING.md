# In-game test checklist

Run through this list before tagging a release. Note your client build from `/slack status`, and keep BugSack open.

Slacker's logic and the companion are covered by tests outside the game (`make test`). Live texture reloading, Edit Mode, the options panel and layout still need a real client.

## Companion install (each OS)

| Check | Expected result |
|---|---|
| First open | Opening the download installs it (Windows: `%LOCALAPPDATA%\Programs\Slacker`, macOS: `~/Applications`, Linux: `~/.local/bin`), shows a welcome message, and the speech bubble appears in the tray / menu bar. |
| Welcome | The welcome message says whether it found the WoW addon. |
| Menu | It shows the Slack count, "WoW: connected…" (or "addon not found. Get it…", which opens the install page), **Start when I log in** (checked), **Help** and **Quit Slacker**. |
| Count | A Slack DM updates the menu's count. Windows/Linux: the icon gets a red dot. macOS: the count appears beside the icon. |
| Login | After logging out and back in, it's running again. Unchecking **Start when I log in** stops that. |
| Open again | Opening the download again while it runs says it's already running. |
| Install order | Install the companion before the addon. Within about 10 seconds of installing the addon, the menu says connected. |
| Quit | **Quit Slacker** removes the icon, and the in-game icon dims to "companion not running". |

## Bridge

| Check | Expected result |
|---|---|
| Offline | With the companion stopped, the icon is dim and the tooltip says the companion app is needed. Clicking it shows a popup with the download link already selected. About 12 seconds after login, chat says so once. Turning off **Remind me when it isn't running** stops the chat line. |
| Live updates | Run `slacker-companion demo`. The badge steps through 1, 2, 3, 7, 12, 42, 99, 99+ **without** `/reload`. With `/slack debug` on, each read prints the new size. |
| Real notification | Start the companion and minimize Slack. A DM from someone shows a badge within about two seconds. |
| Clear | Focus Slack (Windows) or read the message (macOS, Linux). The badge goes away. |
| Reload | `/reload` while the badge shows a count. It comes back with the same count, without a flash or sound. |
| Clean exit | Stop the companion with Ctrl+C. The icon goes dim. |

## Icon

| Check | Expected result |
|---|---|
| States | No notifications: muted gray bubble. Notifications: full-color bubble with a red badge. 10 and up stretches the badge into a pill. 100 and up shows `99+`. |
| Tooltip | Hover shows `Slack: N notifications`, the time of the last one, and the hint line. Turning **Tooltip** off hides it. |
| New notification | The icon flashes three times. With a sound chosen, it plays once. Neither happens at login or after `/reload`. |
| Drag | While unlocked, dragging moves the icon, and the position survives `/reload`. Locked, dragging does nothing. |
| Edit Mode | **Move** opens Edit Mode with a selection box labeled "Slacker" and a sample badge. Dragging saves. Selecting it shows Icon size, Badge size, Opacity and More options. Switching Edit Mode layouts moves the icon to that layout's spot. |
| Sizes | Icon size, Badge size and Opacity apply live from both the options panel and Edit Mode, and the two stay in sync. |
| Right-click | Opens the options panel. |
| Combat | Hover, drag and badge updates in combat raise no errors or taint. |

## Options and commands

| Check | Expected result |
|---|---|
| Panel | Esc → Options → AddOns → Slacker lists every setting. **Defaults** restores them. |
| Sound | Picking a sound in the dropdown plays it. SharedMedia sounds show up when installed. |
| Test | **Test** and `/slack test` cycle sample counts, then hand back to the real count. |
| Compartment | The minimap addons menu entry opens options on left-click and tests on right-click. |
| Commands | `/slack`, `/slack move`, `lock`, `unlock`, `reset`, `status` and `debug` all work, and an unknown command prints the list. |

## Needs confirming on a live client

- The client re-reads `Bridge\count.tga` after the probe texture is cleared and set again. This is the whole live-update mechanism. If it caches the file, only `/reload` updates the badge.
- `Texture:GetSize()` on the unsized, single-anchor probe returns the file's native size once `IsObjectLoaded()` is true, including while its alpha is 0.
- 8-bit grayscale TGA loads, at sizes up to 128×1024.
- LibEditMode lists Interface 16001 as supported (PoisonSlot uses it on this client). If it errors, Slacker falls back to dragging.
