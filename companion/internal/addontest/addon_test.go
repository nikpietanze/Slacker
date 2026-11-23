// Package addontest loads the real WoW addon into a Lua 5.1 VM with a mocked
// UI API and checks it decodes the bridge files the companion writes.
package addontest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"slacker/companion/internal/bridge"
)

// The addon lives at the repo root.
var addonSrc = filepath.Join("..", "..", "..")

// newAddon copies the addon into a temp "AddOns" folder, loads the files its
// TOC lists (skipping bundled Libs, so Edit Mode support is off), and logs in.
func newAddon(t *testing.T) (*lua.LState, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), bridge.AddonName)
	toc, err := os.ReadFile(filepath.Join(addonSrc, bridge.AddonName+".toc"))
	must(t, err)
	var files []string
	for _, line := range strings.Split(string(toc), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, `Libs\`) {
			continue
		}
		files = append(files, line)
	}
	for _, f := range append([]string{bridge.AddonName + ".toc", "Media", "Bridge"}, files...) {
		copyPath(t, filepath.Join(addonSrc, f), filepath.Join(dir, f))
	}

	L := lua.NewState()
	t.Cleanup(L.Close)
	// tga_size(root, [[Interface\AddOns\Slacker\Bridge\count.tga]]) -> w, h
	L.SetGlobal("tga_size", L.NewFunction(func(L *lua.LState) int {
		root, p := L.CheckString(1), L.CheckString(2)
		prefix := `Interface\AddOns\` + bridge.AddonName + `\`
		rel := strings.ReplaceAll(strings.TrimPrefix(p, prefix), `\`, string(filepath.Separator))
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || len(b) < 18 {
			L.Push(lua.LNumber(0))
			L.Push(lua.LNumber(0))
			return 2
		}
		L.Push(lua.LNumber(int(b[12]) | int(b[13])<<8))
		L.Push(lua.LNumber(int(b[14]) | int(b[15])<<8))
		return 2
	}))
	must(t, L.DoFile(filepath.Join("testdata", "wow_mock.lua")))
	must(t, L.DoString(`mock.root = [[`+dir+`]]`))

	// Load each file with WoW's (addonName, namespace) varargs.
	ns := L.NewTable()
	for _, f := range files {
		fn, err := L.LoadFile(filepath.Join(dir, f))
		must(t, err)
		L.Push(fn)
		L.Push(lua.LString(bridge.AddonName))
		L.Push(ns)
		must(t, L.PCall(2, 0, nil))
	}
	L.SetGlobal("ns", ns)
	run(t, L, `mock.fire("ADDON_LOADED", "Slacker"); mock.fire("PLAYER_LOGIN"); mock.advance(1)`)
	return L, dir
}

func run(t *testing.T, L *lua.LState, code string) {
	t.Helper()
	must(t, L.DoString(code))
}

func eval(t *testing.T, L *lua.LState, expr string) string {
	t.Helper()
	must(t, L.DoString(`__r = tostring(`+expr+`)`))
	return L.GetGlobal("__r").String()
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func copyPath(t *testing.T, src, dst string) {
	t.Helper()
	must(t, filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		os.MkdirAll(filepath.Dir(target), 0o755)
		return os.WriteFile(target, b, 0o644)
	}))
}

func itoa(n int) string {
	return strings.TrimSpace(lua.LNumber(n).String())
}

func TestLuaCodecMatchesGo(t *testing.T) {
	L, _ := newAddon(t)
	for code := 0; code <= bridge.CodeForCount(1000); code++ {
		w, h := bridge.Dimensions(code)
		got := eval(t, L, "ns.DecodeSize("+itoa(w)+","+itoa(h)+")")
		want := "false"
		if code != bridge.CodeOffline {
			want = itoa(code - 1)
		}
		if got != want {
			t.Fatalf("Lua decodes %dx%d as %s, Go says %s", w, h, got, want)
		}
	}
	if got := eval(t, L, "ns.DecodeSize(3, 5)"); got != "nil" {
		t.Errorf("non-bridge size decoded as %s", got)
	}
}

func TestBadgeFollowsCompanion(t *testing.T) {
	L, dir := newAddon(t)
	if got := eval(t, L, "ns.count"); got != "false" {
		t.Fatalf("shipped placeholder should read as offline, got %s", got)
	}
	run(t, L, `SlackerDB.sound = "kit:TELL_MESSAGE"`)
	steps := []struct {
		count   int
		badge   string
		shown   string
		desat   string
		flashes string
		sounds  string
	}{
		{0, "", "false", "true", "nil", "0"},    // companion starts: no badge, muted
		{3, "3", "true", "false", "1", "1"},     // new notifications: badge, flash, sound
		{12, "12", "true", "false", "2", "2"},   // more
		{150, "99+", "true", "false", "3", "3"}, // capped
		{0, "99+", "false", "true", "3", "3"},   // cleared (text stays, badge hidden)
		{2, "2", "true", "false", "4", "4"},     // new again after clearing
	}
	for _, s := range steps {
		must(t, bridge.WriteCode(dir, bridge.CodeForCount(s.count)))
		run(t, L, "mock.advance(1.5)")
		if got := eval(t, L, "ns.count"); got != itoa(min(s.count, bridge.MaxCount+1)) {
			t.Fatalf("count %d: addon reads %s", s.count, got)
		}
		if s.badge != "" {
			if got := eval(t, L, "badgeText(ns)"); got != s.badge {
				t.Errorf("count %d: badge text %q, want %q", s.count, got, s.badge)
			}
		}
		for _, c := range []struct{ expr, want string }{
			{"badgeShown(ns)", s.shown},
			{"iconDesaturated(ns)", s.desat},
			{"flashes(ns)", s.flashes},
			{"mock.sounds", s.sounds},
		} {
			if got := eval(t, L, c.expr); got != c.want {
				t.Errorf("count %d: %s = %s, want %s", s.count, c.expr, got, c.want)
			}
		}
	}
	must(t, bridge.WriteCode(dir, bridge.CodeOffline))
	run(t, L, "mock.advance(1.5)")
	if got := eval(t, L, "ns.count"); got != "false" {
		t.Fatalf("offline: addon reads %s", got)
	}
}

func TestSlashCommands(t *testing.T) {
	L, _ := newAddon(t)
	run(t, L, `
		SlashCmdList.SLACKER("lock")
		assert(SlackerDB.locked == true, "lock")
		SlashCmdList.SLACKER("  UNLOCK ")
		assert(SlackerDB.locked == false, "unlock")
		SlashCmdList.SLACKER("")
		assert(mock.openedOptions == "Slacker", "options open")
		SlashCmdList.SLACKER("status")
		local found
		for _, line in ipairs(mock.printed) do
			if line:find("dev on client 1.60.1", 1, true) then found = true end
		end
		assert(found, "status prints version and client")
		SlashCmdList.SLACKER("setup")
		assert(mock.popup == "SLACKER_SETUP", "setup opens the link popup")
		assert(mock.popupText == ns.COMPANION_URL, "popup shows the download link")
		assert(mock.popupTextAfterTyping == ns.COMPANION_URL, "typing can't change the link")
		assert(mock.popupHidden, "escape closes the popup")
		SlashCmdList.SLACKER("bogus")
		SlashCmdList.SLACKER("debug")
		SlashCmdList.SLACKER("test")
		mock.advance(2)
		assert(ns.count == 3, "test mode shows fake counts, got " .. tostring(ns.count))
		mock.advance(8)
		assert(ns.count == false, "test mode hands back to the bridge, got " .. tostring(ns.count))
		Slacker_OnCompartmentClick(nil, "LeftButton")
		Slacker_OnCompartmentEnter(nil, ns.button)
		Slacker_OnCompartmentLeave()
	`)
}

func TestSettingsAndPosition(t *testing.T) {
	L, _ := newAddon(t)
	run(t, L, `
		-- every option is registered with the native settings panel
		for _, key in ipairs({ "locked", "tooltip", "iconSize", "badgeScale", "opacity", "flash", "sound" }) do
			assert(mock.settings[key], key .. " not registered")
		end
		assert(#mock.lastDropdown >= 5, "sound dropdown lists None plus built-in sounds")

		mock.settings.iconSize:SetValue(48)
		assert(ns.button:GetWidth() == 48, "icon size applies")
		mock.settings.opacity:SetValue(50)
		assert(math.abs(ns.button:GetAlpha() - 0.5 * 0.45) < 0.001, "opacity applies (offline is dimmed)")

		for _, click in pairs(mock.buttons) do click() end -- Move, Reset, Test

		-- dragging saves an offset from the screen centre (mock centre 512,384)
		ns.button._scripts.OnDragStop(ns.button)
		local p = SlackerDB.positions.default
		assert(p.point == "CENTER" and p.x == -12 and p.y == 16, "drag position saved")
		ns.ResetPosition()
		assert(SlackerDB.positions.default.y == ns.DEFAULT_POSITION.y, "reset position")

		ns.ResetSettings()
		assert(SlackerDB.iconSize == 32 and SlackerDB.sound == "none", "defaults restored")
	`)
}

func TestCompanionMissingHelp(t *testing.T) {
	L, _ := newAddon(t)
	run(t, L, `
		assert(ns.count == false, "starts offline (shipped placeholder)")
		mock.advance(12)
		local reminded
		for _, line in ipairs(mock.printed) do
			if line:find("/slack setup", 1, true) then reminded = true end
		end
		assert(reminded, "login reminder when the companion isn't running")
		mock.popup = nil
		ns.button._scripts.OnClick(ns.button, "LeftButton")
		assert(mock.popup == "SLACKER_SETUP", "clicking the offline icon offers the download link")
		mock.popup = nil
		ns.button._scripts.OnClick(ns.button, "RightButton")
		assert(mock.popup == nil and mock.openedOptions == "Slacker", "right-click still opens options")
	`)
}

func TestNoReminderWhenConnected(t *testing.T) {
	L, dir := newAddon(t)
	must(t, bridge.WriteCode(dir, bridge.CodeForCount(0)))
	run(t, L, `
		mock.advance(12)
		for _, line in ipairs(mock.printed) do
			assert(not line:find("/slack setup", 1, true), "no reminder while connected")
		end
		mock.popup = nil
		ns.button._scripts.OnClick(ns.button, "LeftButton")
		assert(mock.popup == nil, "left-click does nothing while connected")
	`)
}
