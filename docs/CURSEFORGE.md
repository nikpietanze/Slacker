A tiny icon that tells you when you have Slack notifications, without leaving the game.

## Features

- **One small icon.** Muted when Slack has nothing for you. A red badge with the count when it does.
- **Hover for details.** "Slack: 3 notifications" and the time of the last one.
- **Flash and sound.** A quick flash on new notifications, plus an optional sound.
- **Edit Mode support.** Move and resize it like any Blizzard frame.
- **Private.** No Slack login or app install. Nothing leaves your computer.

## Needs the companion app

WoW addons can't see other programs, so a small companion app (Windows, macOS, Linux) watches the notifications Slack already shows and passes the count to the addon. Download it from [GitHub Releases](https://github.com/nikpietanze/Slacker/releases/latest).

## Commands

| Command | What it does |
|---|---|
| `/slack` | Options |
| `/slack test` | Preview the badge |
| `/slack move` | Move the icon |
| `/slack reset` | Reset its position |
| `/slack status` | Info for bug reports |

## Bugs and ideas

[GitHub Issues](https://github.com/nikpietanze/Slacker/issues). Include `/slack status` and any Lua errors.
