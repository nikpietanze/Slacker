-- A deliberately small fake of the WoW UI API: just enough to load the addon,
-- drive its timers and OnUpdate scripts, and inspect what it shows.
-- Texture:GetSize() on an unsized texture returns the real size of the file on
-- disk (via the Go helper tga_size), like the game does.

local clock = 0
local timers = {}
local frames = {}
mock = { printed = {}, sounds = 0, root = nil }

function GetTime() return clock end
function time() return 1700000000 + math.floor(clock) end
function date(fmt, t) return "12:34" end

C_Timer = {}
function C_Timer.After(delay, fn)
	timers[#timers + 1] = { at = clock + delay, fn = fn }
end
function C_Timer.NewTicker(interval, fn)
	local t = { cancelled = false }
	local function tick()
		if t.cancelled then return end
		fn()
		C_Timer.After(interval, tick)
	end
	C_Timer.After(interval, tick)
	function t:Cancel() self.cancelled = true end
	return t
end

-- Advance the fake clock in 1/30 s frames, firing timers and OnUpdate.
function mock.advance(seconds)
	local stop = clock + seconds
	while clock < stop do
		clock = clock + 1 / 30
		local due = {}
		for i = #timers, 1, -1 do
			if timers[i].at <= clock then
				due[#due + 1] = timers[i]
				table.remove(timers, i)
			end
		end
		table.sort(due, function(a, b) return a.at < b.at end)
		for _, t in ipairs(due) do t.fn() end
		for _, f in ipairs(frames) do
			local s = f._scripts.OnUpdate
			if s then s(f, 1 / 30) end
		end
	end
end

local function noop() end

local function newObject(kind)
	local o = {
		_kind = kind, _scripts = {}, _shown = true, _alpha = 1, _scale = 1,
		_w = 0, _h = 0, _points = {}, _text = "", _checked = false, _events = {},
	}
	return setmetatable(o, { __index = function(t, k)
		local m = mock.methods[k]
		if m then return m end
		if type(k) == "string" and k:sub(1, 1) == "_" then return nil end -- mock state
		return noop
	end })
end

mock.methods = {
	SetScript = function(self, name, fn) self._scripts[name] = fn end,
	GetScript = function(self, name) return self._scripts[name] end,
	SetSize = function(self, w, h) self._w, self._h, self._sized = w, h, true end,
	SetWidth = function(self, w) self._w, self._sized = w, true end,
	SetHeight = function(self, h) self._h, self._sized = h, true end,
	GetSize = function(self)
		if self._kind == "Texture" and not self._sized then
			if self._file then return tga_size(mock.root, self._file) end
			return 0, 0
		end
		return self._w, self._h
	end,
	GetWidth = function(self) return self._w end,
	GetHeight = function(self) return self._h end,
	SetPoint = function(self, p, rel, rp, x, y)
		if type(rel) == "number" then rel, rp, x, y = nil, p, rel, rp end
		self._points[#self._points + 1] = { p, rel, rp or p, x or 0, y or 0 }
	end,
	ClearAllPoints = function(self) self._points = {} end,
	GetPoint = function(self, i)
		local pt = self._points[i or 1]
		if not pt then return nil end
		return pt[1], pt[2], pt[3], pt[4], pt[5]
	end,
	GetCenter = function(self) return 500, 400 end,
	GetEffectiveScale = function(self) return self._scale end,
	GetScale = function(self) return self._scale end,
	SetScale = function(self, s) self._scale = s end,
	SetAlpha = function(self, a) self._alpha = a end,
	GetAlpha = function(self) return self._alpha end,
	Show = function(self) self._shown = true; if self._scripts.OnShow then self._scripts.OnShow(self) end end,
	Hide = function(self) self._shown = false end,
	SetShown = function(self, v) self._shown = not not v end,
	IsShown = function(self) return self._shown end,
	SetTexture = function(self, f) self._file = f end,
	IsObjectLoaded = function(self) return true end,
	SetDesaturated = function(self, v) self._desat = v end,
	SetText = function(self, t) self._text = t end,
	GetText = function(self) return self._text end,
	GetStringWidth = function(self) return #tostring(self._text) * 6 end,
	GetFont = function(self) return "Fonts\\FRIZQT__.TTF", 12, "" end,
	GetChecked = function(self) return self._checked end,
	SetChecked = function(self, v) self._checked = v end,
	GetFrameLevel = function(self) return 1 end,
	IsOwned = function(self, f) return false end,
	CreateTexture = function(self) return newObject("Texture") end,
	CreateFontString = function(self) return newObject("FontString") end,
	CreateAnimationGroup = function(self)
		local g = newObject("AnimationGroup")
		g.Play = function(g) g._played = (g._played or 0) + 1 end
		g.CreateAnimation = function() return newObject("Animation") end
		return g
	end,
	RegisterEvent = function(self, e) self._events[e] = true end,
	UnregisterEvent = function(self, e) self._events[e] = nil end,
	StartMoving = noop,
	StopMovingOrSizing = noop,
}

function CreateFrame(kind, name, parent)
	local f = newObject(kind)
	f._parent = parent
	frames[#frames + 1] = f
	if name then _G[name] = f end
	return f
end

function mock.fire(event, ...)
	for _, f in ipairs(frames) do
		if f._events[event] and f._scripts.OnEvent then
			f._scripts.OnEvent(f, event, ...)
		end
	end
end

UIParent = newObject("Frame")
UIParent.GetHeight = function() return 768 end
UIParent.GetCenter = function() return 512, 384 end
GameTooltip = newObject("GameTooltip")
GameFontNormal = newObject("Font")
STANDARD_TEXT_FONT = "Fonts\\FRIZQT__.TTF"
DEFAULT_CHAT_FRAME = { AddMessage = function(_, m) mock.printed[#mock.printed + 1] = m end }
SOUNDKIT = { TELL_MESSAGE = 3081 }
function PlaySound() mock.sounds = mock.sounds + 1 end
UISpecialFrames = {}
SlashCmdList = {}
tinsert = table.insert

-- Inspectors for the addon's UI (exposed as ns.ui for tests and debugging).
function badgeText(ns) return ns.ui.badgeText:GetText() end
function badgeShown(ns) return ns.ui.badge:IsShown() end
function iconDesaturated(ns) return ns.ui.icon._desat end
function flashes(ns) return ns.ui.glowAnim._played end

-- Click every button whose ancestor is root (settings panel smoke test).
function mock.clickAll(root)
	local n = 0
	for _, f in ipairs(frames) do
		local p = f._parent
		while p and p ~= root do p = p._parent end
		if p == root and f._scripts.OnClick then
			if f._scripts.OnEnter then f._scripts.OnEnter(f) end
			f._scripts.OnClick(f, "LeftButton")
			if f._scripts.OnLeave then f._scripts.OnLeave(f) end
			n = n + 1
		end
	end
	return n
end

-- Blizzard globals used by the scaffold patterns.
C_AddOns = { GetAddOnMetadata = function(_, field) return field == "Version" and "@project-version@" or nil end }
function CopyTable(t) local c = {} for k, v in pairs(t) do c[k] = type(v) == "table" and CopyTable(v) or v end return c end
function wipe(t) for k in pairs(t) do t[k] = nil end return t end
function strtrim(s) return (s:gsub("^%s+", ""):gsub("%s+$", "")) end
function print(...) local parts = {} for i = 1, select("#", ...) do parts[#parts + 1] = tostring(select(i, ...)) end mock.printed[#mock.printed + 1] = table.concat(parts, " ") end
function GetBuildInfo() return "1.60.1", "12345", "Nov 1 2026", 16001 end
function InCombatLockdown() return false end

-- Native settings API: records settings, callbacks and buttons for tests.
mock.settings, mock.callbacks, mock.buttons = {}, {}, {}
Settings = { VarType = { Boolean = "boolean", Number = "number", String = "string" } }
function Settings.RegisterVerticalLayoutCategory(name)
	return { GetID = function() return name end }, { AddInitializer = function() end }
end
function Settings.RegisterAddOnSetting(_, variable, key, tbl, _, _, default)
	local s = { default = default }
	function s:SetValue(v)
		tbl[key] = v
		local cb = mock.callbacks[variable]
		if cb then cb(nil, s, v) end
	end
	function s:GetValue() return tbl[key] end
	mock.settings[key] = s
	return s
end
function Settings.SetOnValueChangedCallback(variable, cb) mock.callbacks[variable] = cb end
function Settings.CreateCheckbox() return {} end
function Settings.CreateSlider() return {} end
function Settings.CreateSliderOptions()
	return { SetLabelFormatter = function(self, _, fn) self.format = fn end }
end
function Settings.CreateControlTextContainer()
	local data = {}
	return { Add = function(_, value, text) data[#data + 1] = { value = value, text = text } end,
		GetData = function() return data end }
end
function Settings.CreateDropdown(_, _, options) mock.lastDropdown = options() return {} end
function Settings.RegisterAddOnCategory() end
function Settings.OpenToCategory(id) mock.openedOptions = id end
MinimalSliderWithSteppersMixin = { Label = { Right = 1 } }
CLOSE = "Close"
StaticPopupDialogs = {}
function StaticPopup_Show(which)
	mock.popup = which
	local info = StaticPopupDialogs[which]
	local box = newObject("EditBox")
	box.HighlightText = function() end
	box.SetFocus = function() end
	local dialog = newObject("Frame")
	dialog.editBox = box
	box.GetParent = function() return dialog end
	if info.OnShow then info.OnShow(dialog) end
	mock.popupText = box:GetText()
	box:SetText("oops")
	if info.EditBoxOnTextChanged then info.EditBoxOnTextChanged(box) end
	mock.popupTextAfterTyping = box:GetText()
	if info.EditBoxOnEscapePressed then info.EditBoxOnEscapePressed(box) end
	mock.popupHidden = not dialog:IsShown()
end
function CreateSettingsListSectionHeaderInitializer() return {} end
function CreateSettingsButtonInitializer(name, _, onClick) mock.buttons[name] = onClick return {} end
