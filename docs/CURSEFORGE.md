A tiny icon that tells you when you have unread Slack notifications, without leaving the game.

**Needs a free companion app** (Windows, macOS, Linux). WoW addons can't see other programs, so a small app on your computer tells Slacker what Slack is showing. Install the addon, click the Slacker icon in game for the download link, and open the app once. That's it.

## Features

- **One small icon.** Muted when Slack has nothing for you, with a red badge and the count (up to 99+) when it does.
- **Hard to miss.** A quick flash when a new notification arrives, then a gentle red pulse until you've read them. An optional sound plays too, built-in or from SharedMedia.
- **Fast.** New notifications show up within a couple of seconds, with no `/reload`.
- **Hover for details.** "Slack: 3 notifications" and the time of the last one.
- **Edit Mode support.** Move and resize it like any Blizzard frame, with a position saved for each Edit Mode layout.
- **Private.** No Slack login, no workspace app, no admin approval. Nothing leaves your computer.

## Setup

1. Install Slacker.
2. Get the companion app from [GitHub](https://github.com/nikpietanze/Slacker/releases/latest), or click the Slacker icon in game for the link.
3. Open it once. It installs itself, starts when you log in, and sits in your system tray or menu bar.

## Commands

| Command | What it does |
|---|---|
| `/slacker` | Options |
| `/slacker setup` | Companion download link |
| `/slacker test` | Preview the badge |
| `/slacker move` | Move the icon |
| `/slacker reset` | Reset its position |
| `/slacker status` | Info for bug reports |

## Good to know

- The companion only reads what Slack already shows on your computer: Windows notifications, the macOS Dock badge, or Linux desktop notifications. On Windows, keep Slack's notifications going through Windows (Slack → Settings → Notifications), which is the default.
- The first time you open the companion, Windows or macOS may warn you because it isn't code-signed yet. Choose **Run anyway** (Windows) or **Open Anyway** in Privacy & Security (macOS).
- Clicking the icon can't switch you to Slack: addons can't open other programs.

## Bugs and ideas

[GitHub Issues](https://github.com/nikpietanze/Slacker/issues). Include `/slacker status` and any Lua errors.
