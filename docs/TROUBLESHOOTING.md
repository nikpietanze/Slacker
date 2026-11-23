# Troubleshooting

Start with:

```
slacker-companion doctor
```

It shows the notification source, what it can see right now, which addon folders it will write to, and what the bridge file currently says.

In game, `/slack status` shows what the addon last read, and `/slack debug` prints every bridge read to chat.

## I don't see the speech bubble in my tray or menu bar

- **Windows:** it may be in the overflow area. Click the **^** arrow next to the clock. Drag it onto the taskbar to keep it visible.
- **macOS:** a crowded menu bar can hide icons behind the notch. Quit an app or two, or use a menu bar manager.
- **Linux:** GNOME needs the *AppIndicator and KStatusNotifierItem Support* extension (Ubuntu ships it). Slacker works the same without the icon.
- **Not running at all:** open the app again. It also starts by itself when you log in.

## Windows: "Windows protected your PC"

The download isn't code-signed yet, so SmartScreen doesn't recognize it. Click **More info → Run anyway**. You only see this once.

## macOS: "Slacker Companion can't be opened" or "Apple could not verify…"

The app isn't notarized by Apple yet. Open **System Settings → Privacy & Security**, scroll down and click **Open Anyway**. You only need to do this once.

## The icon says "companion not running"

- The companion isn't running, or it can't find your addon folder. Open its tray / menu bar menu: it says whether it found the WoW addon. For details, run `doctor`.
- It writes to every `World of Warcraft/_*_/Interface/AddOns/Slacker` it finds. If your WoW is somewhere unusual, pass `--wow-dir` (the folder that contains `_forever_`) or `--addon-dir` (the `Slacker` folder itself). If you use autostart, run `autostart on` again with the same flags.
- If you installed the addon after starting WoW, `/reload` once.

## The badge only updates after `/reload`

The live bridge relies on the game client re-reading `Bridge/count.tga` after the addon releases it. Check:

1. Quit the companion from its tray / menu bar icon, run `slacker-companion demo` from a terminal, and watch the in-game icon.
2. In game, `/slack debug` prints `bridge WxH -> value` every second.
   - If the size never changes while `demo` runs, the client is serving a cached copy. Please [open an issue](https://github.com/nikpietanze/Slacker/issues) with your client version (`/dump GetBuildInfo()`) and OS.
   - If it prints `0x0`, the texture isn't loading. Make sure `Bridge/count.tga` exists in the addon folder; it ships with the addon.

`/reload` always picks up the current count, so that is the fallback.

## Windows: the count stays at 0

- In Slack, open **Settings → Notifications** and set **Deliver notifications via** to Windows notifications, not Slack's built-in notifications.
- In Windows, open **Settings → System → Notifications** and check that notifications are on for Slack.
- Slack doesn't notify while you're looking at it. Minimize it before testing.
- `doctor` lists the Slack entries in the notification database. "No Slack entry yet" means Windows has never shown a Slack notification.
- Windows 10 1809 or newer is required (for `winsqlite3.dll`).

## Windows: the count doesn't go down

The count clears when the Slack window is focused, or when you clear Slack's notifications in the notification center (Win+N). Reading Slack on your phone doesn't clear desktop notifications. Open the desktop app once.

## macOS: the count stays at 0

- The companion mirrors Slack's Dock badge. Check that Slack shows a number on its Dock icon. If it doesn't: **System Settings → Notifications → Slack → Badge application icon**.
- A dot (•) on the Dock icon means unread messages that don't notify you. Those count as 0 by design.

## Linux: the count stays at 0 or never clears

- `doctor` must show `bus monitoring: ok`. If it fails, your D-Bus policy blocks monitoring. Please open an issue with your distro.
- Flatpak and Snap Slack builds may report a different app name. Run the companion with `-v` and look for `slack notification requested` when a notification appears.
- Notifications that only time out stay counted, as they do in your notification tray. Dismiss them, or open the channel in Slack (Slack closes its own notifications as you read).

## The addon shows "Out of date"

Tick *Load out of date AddOns* on the character-select AddOns screen, or set the `## Interface:` line in `Slacker.toc` to the number from `/dump select(4, GetBuildInfo())`.

## Uninstall

Turn off **Start when I log in** in the companion's menu, choose **Quit Slacker**, then delete the app:

- **Windows:** `%LOCALAPPDATA%\Programs\Slacker`
- **macOS:** `~/Applications/Slacker Companion.app`
- **Linux:** `~/.local/bin/slacker-companion`

Then delete the `Slacker` addon folder. The settings live in `WTF/Account/<account>/SavedVariables/Slacker.lua`.
