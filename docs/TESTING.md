# In-game test checklist

Run through this list before tagging a release. Note your client build from `/slacker status`, and keep BugSack open.

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
| Live updates | Run `slacker-companion demo`. The badge steps through 1, 2, 3, 7, 12, 42, 99, 99+ **without** `/reload`. With `/slacker debug` on, each read prints the new size. |
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
| Test | **Test** and `/slacker test` cycle sample counts, then hand back to the real count. |
| Compartment | The minimap addons menu entry opens options on left-click and tests on right-click. |
| Commands | `/slacker`, `/slacker move`, `lock`, `unlock`, `reset`, `status` and `debug` all work, and an unknown command prints the list. |

## Learned on a live client

On WoW Forever 1.60.1 (build 71037, macOS, Nov 2026):

- `Texture:GetSize()` on an unsized, single-anchor, alpha-0 texture returns the file's exact pixel size once loaded (2×2 read as `2.00x2.00`, the 64×64 icon as `64.00x64.00`).
- 32-bit TGA loads. An 8-bit grayscale TGA did **not** (it measured 1×1).
- Textures stay cached while in use, and unused ones are freed in a cleanup that ran 7 to 76 seconds apart. With one bridge file read every second, the badge picked up a change only after a cleanup, and got stuck when the file was held most of the time. Files left unused were read fresh after a cleanup. That's why the bridge is a ring of 128 slots.
- LibEditMode works on Interface 16001 (PoisonSlot uses it too).

## Still to confirm

- The slot ring in game: changes should show within about two seconds. After a test, `bridgeLog` in `SavedVariables/Slacker.lua` lists when each change arrived.
- Windows and Linux (Wine) clients, which should behave the same.
