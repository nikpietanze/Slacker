# Engineering Handoff: WoW Slack Notification Indicator

**Status:** v1 implemented. Live texture reloading still needs confirming on a real client (see [TESTING.md](TESTING.md)).
**Target:** World of Warcraft: Forever (Interface 16001), plus a local companion app for Windows, macOS and Linux.
**Design principle:** *Do not recreate Slack.* The only question the UI answers is "Do I need to check Slack?"

## Goal

Build a very small WoW Forever addon and a local companion app that show whether the user has unread Slack notifications while playing WoW.

## Desired in-game UX

A small movable icon that can be positioned anywhere on screen.

Default appearance:

- A small speech-bubble icon with an **S** inside.
- Draggable when unlocked.
- Position persists between sessions.
- Minimal visual footprint.

Notification behavior:

- No notifications → normal/muted icon with no badge.
- Notifications present → a **red circular badge** in the top-right corner.
- The badge shows the unread notification count, capped at `99+`.
- Hover tooltip: `Slack: X notifications`.
- Optional click action: focus/open the Slack desktop app. *(Not possible: addons can't launch or focus other programs.)*
- Right-click or the slash command opens basic settings.

Settings:

- Lock/unlock position.
- Show/hide tooltip.
- Enable/disable notification sound.
- Badge scale.
- Icon scale.
- Reset position.

## Architecture

### 1. WoW addon

Pure Lua addon for WoW Forever. It:

- Renders the icon and the notification badge.
- Persists settings and frame position.
- Receives the current notification count from the local companion bridge, and updates immediately when it changes.
- Holds no Slack credentials and never communicates with Slack.

The addon only needs a simple state such as `notificationCount = 3`.

### 2. Desktop companion app

It:

- Monitors Slack desktop notifications through the operating system's notification system.
- Does **not** use Slack OAuth or the Slack API, and doesn't require a Slack bot, workspace app, token or admin approval.
- Maintains a current notification count.
- Passes that count to the addon through a local bridge compatible with WoW Forever.

## Privacy requirement

The implementation must **not establish a separate connection to Slack**:

`Slack Desktop → OS notifications → Local Companion → WoW`

- Slack sees only normal Slack desktop usage.
- No WoW status, process information or game activity is sent to Slack.
- No notification data is sent to any external server. All processing stays local.

## Notification scope

v1 mirrors whatever notifications Slack already sends through the operating system: direct messages, mentions, and channels the user has set to notify. It does not try to work out unread state on its own through API calls. It reflects Slack's existing notification policy.

## v1 behavior

1. A Slack desktop notification arrives, and the companion detects it.
2. The notification count goes up.
3. The badge in WoW updates.
4. When Slack is opened or focused, or the notifications are cleared, the companion clears or updates the count, and the badge follows.

## Nice-to-haves

- Different badge state for DMs vs mentions.
- Optional notification sound in WoW. *(Done.)*
- Clicking the icon focuses Slack. *(Not possible from the addon sandbox.)*
- Flash/glow the icon briefly when a new notification arrives. *(Done.)*
- Configurable icon opacity. *(Done.)*
- Optional timestamp of the last notification. *(Done, in the tooltip.)*

## Technical priorities

1. Reliable notification detection.
2. Minimal privacy footprint.
3. Tiny, polished in-game UI.
4. No Slack API or admin setup.
5. Open-source codebase with clear separation between addon and companion app.
6. No protected-action or combat issues in WoW.
7. Negligible CPU and resource use.

## Success criteria

The user can:

1. Install the companion app.
2. Install the WoW addon.
3. Receive a Slack desktop notification.
4. Immediately see a red notification-count badge appear on the movable WoW icon.
5. Clear or check Slack and have the badge reset.
6. Do all of this without installing or authorizing a Slack workspace application.

## Non-goals

- Message previews, channel names, sender names or any Slack content.
- Connecting to Slack directly (API, OAuth, bots, tokens).
- Sending anything off the user's computer.
- Recreating any part of Slack's UI in game.
