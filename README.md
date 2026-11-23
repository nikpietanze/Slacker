<p align="center">
  <img src="docs/logo/logo.png" alt="Slacker logo" width="160">
</p>

<h1 align="center">Slacker</h1>

<p align="center">
  A tiny World of Warcraft: Forever addon that answers one question while you play: <b>do I need to check Slack?</b>
</p>

<p align="center">
  <a href="https://github.com/nikpietanze/Slacker/releases/latest"><img src="https://img.shields.io/github/v/release/nikpietanze/Slacker?color=7468f2" alt="Latest release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/nikpietanze/Slacker?color=b48352" alt="MIT license"></a>
</p>

<p align="center">
  <img src="docs/icon-states.png" alt="Muted icon with no notifications, a badge showing 3, and a badge showing 99+" width="640">
</p>

- **One small icon.** A speech bubble with an **S** that you can put anywhere. It's muted while Slack has nothing for you.
- **A badge when it matters.** A red badge shows how many Slack notifications you haven't dealt with, up to `99+`. Hover for `Slack: 3 notifications` and the time of the last one.
- **Clears itself.** Read Slack (or clear its notifications) and the badge goes away.
- **Edit Mode support.** Move and size the icon in Blizzard's Edit Mode, with a position saved for each Edit Mode layout.

No message previews, no channel names, no senders. Only a number.

## How it gets the count

A small **companion app** runs on your computer and watches the notifications Slack already shows through your operating system. It hands the count to the addon.

```
Slack desktop ──► OS notifications ──► companion (local) ──► Slacker addon
```

- **No Slack connection.** The companion never talks to Slack: no tokens, no OAuth, no bot, no workspace app, no admin approval.
- **No network at all.** Nothing about Slack or WoW leaves your computer, and nothing is sent to Slack.

WoW addons can't open sockets or read files, so the companion writes a tiny blank image into the addon's folder, and the image's width and height encode the count. The addon reloads it about once a second. The details are in [docs/HOW-IT-WORKS.md](docs/HOW-IT-WORKS.md).

| Your computer | What the companion watches | When the badge clears |
|---|---|---|
| Windows 10/11 | Slack's notifications in the Windows notification center | When you focus Slack, or the notifications are cleared |
| macOS | Slack's Dock badge (its count of DMs, mentions and keywords) | As you read, when Slack clears its badge |
| Linux | Slack's desktop notifications on the session D-Bus, plus its launcher badge | When Slack closes them, or you dismiss them |

## Options

Everything is under **Esc → Options → AddOns → Slacker** (or type `/slack`, right-click the icon, or click Slacker in the minimap's addons menu):

| Section | Settings |
|---|---|
| Icon | Lock position, tooltip, icon size (20 to 64 px), badge size, opacity, plus Move and Reset |
| New notifications | Flash, sound (built-in or your SharedMedia sounds), plus Test |

The panel's **Defaults** button restores everything. To move the icon, click **Move** (or type `/slack move`) to open Edit Mode, then drag it. Select it there to change its size and opacity. While it's unlocked, you can also just drag it.

## Commands

| Command | What it does |
|---|---|
| `/slack` | Open the options |
| `/slack test` | Show sample counts on the badge for a few seconds |
| `/slack move` | Open Edit Mode to move the icon |
| `/slack unlock` / `/slack lock` | Drag the icon without Edit Mode, then lock it |
| `/slack reset` | Put the icon back in its default spot |
| `/slack status` | Print client, bridge and layout details for bug reports |
| `/slack debug` | Print every read of the companion's bridge file |

`/slacker` works too.

## Install

### 1. The addon

Download the latest `Slacker-*.zip` from [Releases](https://github.com/nikpietanze/Slacker/releases/latest) and extract it into your AddOns folder:

```
World of Warcraft/_forever_/Interface/AddOns/Slacker/Slacker.toc
```

You can also clone this repo into that folder as `Slacker`. Target client: WoW Forever, Interface `16001`.

### 2. The companion

Download the companion for your computer from the same release. It's one file, with nothing to install and almost no CPU or memory use.

**Windows**

1. Download `slacker-companion-windows-amd64.exe` (or `-arm64`), rename it to `slacker-companion.exe`, and put it somewhere permanent, e.g. `%LOCALAPPDATA%\Slacker\`.
2. In Slack, open **Settings → Notifications** and make sure notifications are delivered through **Windows notifications**.
3. Double-click it. A message confirms it's running in the background.
4. Start it at login: run `.\slacker-companion.exe autostart on` in a terminal in that folder.

**macOS**

```sh
chmod +x slacker-companion-darwin-arm64        # -amd64 on Intel Macs
xattr -d com.apple.quarantine slacker-companion-darwin-arm64
./slacker-companion-darwin-arm64 autostart on   # installs a LaunchAgent and starts it
```

Make sure Slack can badge its Dock icon: **System Settings → Notifications → Slack → Badge application icon**.

**Linux** (WoW under Wine, Lutris, Steam/Proton or Bottles)

```sh
chmod +x slacker-companion-linux-amd64
./slacker-companion-linux-amd64 doctor
./slacker-companion-linux-amd64 autostart on --wow-dir "$HOME/Games/battlenet/drive_c/Program Files (x86)/World of Warcraft"
```

The companion finds common install locations on its own. Otherwise, pass `--wow-dir "<folder containing _forever_>"` or `--addon-dir "<…/AddOns/Slacker>"`.

### 3. Check it works

1. `slacker-companion doctor` shows what it can see and which addon folders it found.
2. With the companion stopped and you logged into WoW, run `slacker-companion demo`. The badge should step through 1, 2, 3, 7, 12, 42, 99, 99+.
3. Start the companion again, then have someone DM you.

If the badge only changes after `/reload`, see [Troubleshooting](docs/TROUBLESHOOTING.md#the-badge-only-updates-after-reload).

## Good to know

- **Live updates rely on the game re-reading a changed texture file.** This hasn't been confirmed on a live client yet, and `demo` tells you in a minute. If the client caches the file, the badge still corrects on every `/reload`.
- **Clicking the icon can't switch to Slack.** Addons can't launch or focus other programs.
- **Terms of service.** The addon uses only the public addon API, and the companion only writes a file inside the addon's own folder, much as the WeakAuras Companion does. It doesn't read game memory or send input. Blizzard hasn't commented on this kind of file feed, so use it at your own discretion.

## Bugs and ideas

Open an [issue](https://github.com/nikpietanze/Slacker/issues). For bugs, include the output of `/slack status` and `slacker-companion doctor`, plus any Lua error text (BugSack makes this easy). Pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).

Slacker bundles these libraries unmodified, each under its own license in its folder under `Libs`:

- [LibStub](https://www.wowace.com/projects/libstub) (public domain)
- [LibEditMode](https://github.com/p3lim-wow/LibEditMode) by p3lim (custom license that allows bundling unmodified)

The "S" in the icon is drawn from [Noto Sans](https://fonts.google.com/noto/specimen/Noto+Sans) Bold (SIL Open Font License).

*Not affiliated with Slack Technologies or Blizzard Entertainment.*
