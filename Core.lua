-- Slacker: a tiny icon that shows whether you have unread Slack desktop
-- notifications. The addon never talks to Slack; the companion app on your
-- computer does the watching (see Bridge.lua and docs/HOW-IT-WORKS.md).

local ADDON_NAME, ns = ...

ns.MEDIA = [[Interface\AddOns\]] .. ADDON_NAME .. [[\Media\]]
ns.MAX_COUNT = 99 -- Shown as "99+" above this.
ns.DEFAULT_POSITION = { point = "CENTER", x = 0, y = 200 }
ns.COMPANION_URL = "https://github.com/nikpietanze/Slacker#install"

local DB_VERSION = 1
local DEFAULTS = {
	-- Icon
	locked = false, -- When unlocked, drag the icon to move it (Edit Mode works either way).
	tooltip = true,
	iconSize = 32, -- Pixels.
	badgeScale = 100, -- Percent of the icon's badge size.
	opacity = 100, -- Percent.
	positions = {}, -- Icon position per Edit Mode layout, plus "default".
	-- Alerts
	flash = true, -- A quick flash when a new notification arrives.
	pulse = true, -- Keep a gentle pulse going until they're read.
	sound = "none", -- "none", "kit:<SOUNDKIT key>" or "lsm:<LibSharedMedia name>"
	-- Companion
	remindCompanion = true, -- Say so in chat when the companion isn't running.
	interval = 1, -- Seconds between reads of the companion's bridge file.
}

local loadCallbacks, loginCallbacks = {}, {}
function ns.OnLoad(fn) loadCallbacks[#loadCallbacks + 1] = fn end
function ns.OnLogin(fn) loginCallbacks[#loginCallbacks + 1] = fn end

function ns.Print(message)
	print("|cff8b80f9Slacker|r " .. message)
end

function ns.Version()
	local version = C_AddOns.GetAddOnMetadata(ADDON_NAME, "Version")
	-- Split the token so the packager only substitutes it in the TOC.
	if not version or version == "@project" .. "-version@" then
		return "dev"
	end
	return version
end

local function migrate(db)
	for key, value in pairs(DEFAULTS) do
		if db[key] == nil then
			db[key] = type(value) == "table" and CopyTable(value) or value
		end
	end
	db.version = DB_VERSION
	return db
end

function ns.ResetSettings()
	wipe(ns.db)
	migrate(ns.db)
end

-- Built-in sounds for a new notification. SharedMedia sounds are offered too
-- when an addon providing LibSharedMedia is installed.
ns.SOUNDS = {
	{ key = "kit:TELL_MESSAGE", label = "Whisper" },
	{ key = "kit:MAP_PING", label = "Map ping" },
	{ key = "kit:READY_CHECK", label = "Ready check" },
	{ key = "kit:RAID_WARNING", label = "Raid warning" },
}

function ns.SharedMedia()
	return LibStub and LibStub("LibSharedMedia-3.0", true)
end

function ns.PlaySound(choice)
	if not choice or choice == "none" then return end
	local kind, name = choice:match("^(%a+):(.+)$")
	if kind == "kit" then
		local id = SOUNDKIT and SOUNDKIT[name]
		if id then PlaySound(id, "Master") end
	elseif kind == "lsm" then
		local media = ns.SharedMedia()
		local file = media and media:Fetch("sound", name, true)
		if file then PlaySoundFile(file, "Master") end
	end
end

-- "3 notifications", "99+ notifications", "companion not running" ...
function ns.Describe(count)
	if count == nil then return "checking..." end
	if count == false then return "companion not running" end
	if count > ns.MAX_COUNT then return "99+ notifications" end
	return count .. (count == 1 and " notification" or " notifications")
end

local frame = CreateFrame("Frame")
frame:RegisterEvent("ADDON_LOADED")
frame:RegisterEvent("PLAYER_LOGIN")
frame:SetScript("OnEvent", function(self, event, name)
	if event == "ADDON_LOADED" then
		if name ~= ADDON_NAME then return end
		self:UnregisterEvent("ADDON_LOADED")
		SlackerDB = migrate(SlackerDB or {})
		ns.db = SlackerDB
		for _, fn in ipairs(loadCallbacks) do fn() end
	elseif event == "PLAYER_LOGIN" then
		self:UnregisterEvent("PLAYER_LOGIN")
		for _, fn in ipairs(loginCallbacks) do fn() end
	end
end)

local function printStatus()
	local version, build, _, interface = GetBuildInfo()
	ns.Print(("%s on client %s (%s), interface %s"):format(ns.Version(), version, build, interface))
	print(("  slack: %s"):format(ns.Describe(ns.count)))
	print(("  bridge texture: %s, reference: %s (64x64 file), last read %s"):format(ns.lastSize or "not read yet",
		ns.lastRef or "-", ns.lastRead and (math.floor(GetTime() - ns.lastRead + 0.5) .. "s ago") or "never"))
	local w, h = GetPhysicalScreenSize and GetPhysicalScreenSize()
	print(("  screen: %s, ui scale: %.2f"):format(w and (w .. "x" .. h) or "?", UIParent:GetEffectiveScale()))
	print(("  edit mode: %s, layout: %s"):format(ns.EditModeAvailable() and "yes" or "no", ns.LayoutKey()))
end

-- Addon compartment (the addons button by the minimap).
function Slacker_OnCompartmentClick(_, mouseButton)
	if mouseButton == "RightButton" then
		ns.RunTest()
	else
		ns.OpenOptions()
	end
end

function Slacker_OnCompartmentEnter(_, button)
	GameTooltip:SetOwner(button, "ANCHOR_LEFT")
	GameTooltip:SetText("Slacker")
	GameTooltip:AddLine("Slack: " .. ns.Describe(ns.count), 1, 1, 1)
	GameTooltip:AddDoubleLine("Left-click", "Options", 1, 0.82, 0, 1, 1, 1)
	GameTooltip:AddDoubleLine("Right-click", "Test badge", 1, 0.82, 0, 1, 1, 1)
	GameTooltip:Show()
end

function Slacker_OnCompartmentLeave()
	GameTooltip:Hide()
end

---------------------------------------------------------------------------
-- Companion setup help: the addon can't see Slack without the companion app,
-- so make getting it obvious.
---------------------------------------------------------------------------
local function setupBox(dialog)
	return (dialog.GetEditBox and dialog:GetEditBox()) or dialog.editBox or dialog.EditBox
end

local function hideParent(box)
	local dialog = box:GetParent()
	if dialog then dialog:Hide() end
end

StaticPopupDialogs.SLACKER_SETUP = {
	text = "Slacker needs its free companion app to see Slack.\n\n"
		.. "Copy this link (Ctrl+C), open it in your browser and download the app for your computer. "
		.. "Open it once and you're done: it starts by itself from then on.",
	button1 = CLOSE or "Close",
	hasEditBox = true,
	editBoxWidth = 320,
	OnShow = function(self)
		local box = setupBox(self)
		if box then
			box:SetText(ns.COMPANION_URL)
			box:HighlightText()
			box:SetFocus()
		end
	end,
	EditBoxOnTextChanged = function(box)
		-- Keep the link intact if a stray key lands in the box.
		if box:GetText() ~= ns.COMPANION_URL then
			box:SetText(ns.COMPANION_URL)
			box:HighlightText()
		end
	end,
	EditBoxOnEnterPressed = hideParent,
	EditBoxOnEscapePressed = hideParent,
	timeout = 0,
	whileDead = true,
	hideOnEscape = true,
	preferredIndex = 3,
}

function ns.ShowSetup()
	StaticPopup_Show("SLACKER_SETUP")
end

ns.OnLogin(function()
	-- The first bridge read happens within a second or two of login.
	C_Timer.After(12, function()
		if ns.count == false and ns.db.remindCompanion then
			ns.Print("can't see Slack yet because its companion app isn't running. "
				.. "Click the Slacker icon or type |cffffffff/slacker setup|r to get it.")
		elseif ns.count == nil and ns.lastRead then
			-- Reads happen but never decode: a client quirk worth reporting.
			ns.Print("can't read its bridge file on this client. Please open an issue at "
				.. "github.com/nikpietanze/Slacker with the output of |cffffffff/slacker status|r.")
		end
	end)
end)

local commands = {
	setup = function() ns.ShowSetup() end,
	status = printStatus,
	test = function() ns.RunTest() end,
	move = function() ns.Move() end,
	unlock = function() ns.SetOption("locked", false); ns.Print("Icon unlocked. Drag it to move it.") end,
	lock = function() ns.SetOption("locked", true); ns.Print("Icon locked.") end,
	reset = function() ns.ResetPosition() end,
	debug = function()
		ns.debug = not ns.debug
		ns.Print("Bridge debug " .. (ns.debug and "on." or "off."))
	end,
}

SLASH_SLACKER1 = "/slacker"
SlashCmdList.SLACKER = function(input)
	local command = strtrim((input or ""):lower())
	if command == "" or command == "options" or command == "config" then
		ns.OpenOptions()
	elseif commands[command] then
		commands[command]()
	else
		ns.Print("Commands: /slacker [options | setup | test | move | unlock | lock | reset | status | debug]")
	end
end
