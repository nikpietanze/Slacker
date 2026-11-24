# Changelog

## Unreleased

- Fixed the badge showing old counts: while you stood idle, the game could hand back a copy of the companion's file from minutes earlier, so the badge popped up for notifications you'd already read, and was slow to clear. Slacker now gets a fresh reading every time, and skips any stale copy.
- A new notification now gets a much louder alert: three rounds of a bright flash, a red ring rippling out from the icon, and the bubble bouncing. The slow pulse afterwards stays on as a quieter reminder until you've read it.

## 1.0.0

First release.

- A small speech-bubble icon you can put anywhere. It's muted while you have no Slack notifications, and shows a red badge with the count when you do (up to `99+`).
- Hover for `Slack: 3 notifications` and the time of the last one. The icon flashes when a new notification arrives, keeps a red pulse going until you've read them, and can play a sound (built-in or SharedMedia).
- Blizzard Edit Mode support for moving and sizing the icon, with a position saved for each Edit Mode layout. You can also drag it while it's unlocked.
- Native options panel: lock position, tooltip, icon size, badge size, opacity, flash, pulse until read, sound, plus Move, Reset and Test buttons.
- Addon compartment entry: left-click for options, right-click to test the badge.
- `/slacker status` prints details for bug reports.
- Companion app for Windows, macOS and Linux that mirrors Slack's desktop notifications into the addon. It works locally only, with no Slack login. New notifications reach the badge in about two seconds, with no `/reload`.
- Double-click the companion and it installs itself, starts at login, and shows a tray / menu bar icon with your Slack count, the addon's status, a start-at-login switch and Quit.
- If the companion isn't running, the icon says so, clicking it shows the download link, and a one-time chat reminder after login points you there (`/slacker setup`).
