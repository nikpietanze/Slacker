-- Reads the notification count from the companion app.
--
-- Addons can't read files, but they can load a texture and ask for its size.
-- The companion writes tiny blank images whose sizes encode the count, and
-- this file reads them. See docs/HOW-IT-WORKS.md for the whole story.

local ADDON_NAME, ns = ...

local BRIDGE = [[Interface\AddOns\]] .. ADDON_NAME .. [[\Bridge\]]

---------------------------------------------------------------------------
-- Codec. Mirror of companion/internal/bridge/codec.go.
--
-- The count is split over two images per slot, a tens digit and a ones digit.
-- Each holds a digit 0..10 (10 is a marker): digit d is the d-th exponent
-- pair (a, b), enumerated by diagonals a+b = 0, 1, 2, ..., and the image is
-- 2^(a+1) x 2^(b+1) px.
--   tens 0..9, ones 0..9   count 0..99
--   tens 10,   ones 1      100 or more ("99+")
--   tens 10,   ones 10     companion offline (the placeholders we ship)
--
-- The images are 32-bit: an 8-bit grayscale image measured 1x1 on a real
-- client. As a safeguard against UI scaling, every read also measures a
-- texture of known size (the 64x64 icon), scales by that, and rounds to the
-- nearest power of two.
---------------------------------------------------------------------------
local SLOTS = 128
local MARKER = 10
local REF_SIZE = 64 -- Media\Icon.tga
local LOG2 = math.log(2)
local digitByExp = {}
do
	local n, s = 0, 0
	while n <= MARKER do
		for a = 0, s do
			if n <= MARKER then
				digitByExp[a .. "," .. (s - a)] = n
				n = n + 1
			end
		end
		s = s + 1
	end
end

-- Size in file pixels -> exponent, or nil if it can't be a bridge size.
local function exponent(texels)
	if texels < 1.41 then return nil end -- under 2 px (with rounding slack)
	return math.floor(math.log(texels) / LOG2 + 0.5) - 1
end

-- A measured image size -> digit 0..10, or nil. scaleW/scaleH convert UI
-- units to file pixels (1 when omitted).
function ns.DecodeDigit(w, h, scaleW, scaleH)
	if not w or not h or w <= 0 or h <= 0 then return nil end
	local a, b = exponent(w / (scaleW or 1)), exponent(h / (scaleH or 1))
	if not a or not b then return nil end
	return digitByExp[a .. "," .. b]
end

-- Two digits -> the count (0..100, 100 meaning 99+), false when the companion
-- is offline, or nil for combinations the companion never writes.
function ns.DecodeDigits(tens, ones)
	if tens == nil or ones == nil then return nil end
	if tens == MARKER then
		if ones == MARKER then return false end
		if ones == 1 then return ns.MAX_COUNT + 1 end
		return nil
	end
	if ones == MARKER then return nil end
	return tens * 10 + ones
end

---------------------------------------------------------------------------
-- State
---------------------------------------------------------------------------
ns.count = nil -- nil: not read yet, false: companion offline, number: count
ns.lastNotified = nil -- time() of the most recent increase
local testUntil = 0

local function setCount(value, fromTest)
	if not fromTest and GetTime() < testUntil then return end
	local old = ns.count
	if value == old then return end
	ns.count = value
	local increased = type(value) == "number" and value > (type(old) == "number" and old or 0)
	if increased and old ~= nil then -- not on the first reading after login
		ns.lastNotified = time()
		if ns.OnNewNotification then ns.OnNewNotification() end
	end
	if ns.Refresh then ns.Refresh() end
end

---------------------------------------------------------------------------
-- Poller.
--
-- The client caches every texture it loads, keyed by the file and the wrap
-- and filter modes it was loaded with, and on a real client it can keep an
-- unused entry for many minutes. Loading the same key again returns the old
-- size even after the file changed. So no key is read twice: the companion
-- writes the count into SLOTS identical slots, and we step through every
-- (slot, wrap, wrap, filter) combination, about 13,800 of them, one per
-- second, so nearly four hours pass before any repeats. The position is
-- saved, so a /reload continues where it left off.
--
-- A fresh load takes a frame or more, while a cached one reports loaded at
-- once. Once we've seen a fresh load, a texture that is loaded the moment we
-- set it is a cached (possibly stale) copy: we skip it and try the next key.
--
-- The probes hold a key only for the frame or two it takes to measure it. A
-- value counts only once two reads in a row agree, which skips a slot caught
-- mid-write.
---------------------------------------------------------------------------
local WRAPS = { "CLAMP", "REPEAT", "MIRROR", "CLAMPTOBLACK", "CLAMPTOBLACKADDITIVE", "CLAMPTOWHITE" }
local FILTERS = { "LINEAR", "TRILINEAR", "NEAREST" }
local KEYS = SLOTS * #WRAPS * #WRAPS * #FILTERS
local TRIES = 32 -- keys to try per tick when they turn out to be cached

-- Key number (0-based) -> slot and load modes.
local function keyAt(n)
	local slot = n % SLOTS + 1
	local mode = math.floor(n / SLOTS)
	local h = mode % #WRAPS
	local v = math.floor(mode / #WRAPS) % #WRAPS
	local f = math.floor(mode / (#WRAPS * #WRAPS)) % #FILTERS
	return slot, WRAPS[h + 1], WRAPS[v + 1], FILTERS[f + 1]
end
ns.BridgeKeyAt, ns.BRIDGE_KEYS = keyAt, KEYS

local probeFrame = CreateFrame("Frame", nil, UIParent)
probeFrame:SetSize(1, 1)
probeFrame:SetPoint("BOTTOMLEFT")
local function probeTexture()
	local t = probeFrame:CreateTexture(nil, "BACKGROUND")
	t:SetPoint("TOPLEFT") -- one anchor and no size: GetSize() is the file's size
	t:SetAlpha(0)
	return t
end
local tensProbe, onesProbe, ref = probeTexture(), probeTexture(), probeTexture()

local slot = 0
local waiting, waitStart = false, 0
local candidate -- last decoded value, applied once a second read agrees
local sawFreshLoad = false -- true once a load has taken a frame (so cached ones are telling)
local LOG_SIZE = 30
ns.cachedSkips = 0

-- Keeps the last few changes (with times) in the saved settings, so a bug
-- report can show how quickly the client picked up each one.
local function logChange(value)
	local log = ns.db.bridgeLog or {}
	ns.db.bridgeLog = log
	log[#log + 1] = ("%s key %d %s -> %s"):format(date("%H:%M:%S"), ns.db.bridgeKey or 0, ns.lastSize, tostring(value))
	while #log > LOG_SIZE do table.remove(log, 1) end
end

local function isLoaded(t)
	return not t.IsObjectLoaded or t:IsObjectLoaded()
end

local function loaded(t)
	return isLoaded(t) and (t:GetSize()) > 0
end

local function finishProbe()
	waiting = false
	probeFrame:SetScript("OnUpdate", nil)
	local tw, th = tensProbe:GetSize()
	local ow, oh = onesProbe:GetSize()
	-- Let go right away.
	tensProbe:SetTexture(nil)
	onesProbe:SetTexture(nil)
	local rw, rh = ref:GetSize()
	ns.lastSize = ("%.0fx%.0f+%.0fx%.0f"):format(tw, th, ow, oh)
	ns.lastRef = ("%.2fx%.2f"):format(rw, rh)
	ns.lastRead = GetTime()
	local value
	if rw > 0 and rh > 0 then
		local sw, sh = rw / REF_SIZE, rh / REF_SIZE
		value = ns.DecodeDigits(ns.DecodeDigit(tw, th, sw, sh), ns.DecodeDigit(ow, oh, sw, sh))
	end
	if ns.debug then
		ns.Print(("slot %d: %s, reference %s -> %s"):format(slot, ns.lastSize, ns.lastRef, tostring(value)))
	end
	if value == nil then
		candidate = nil
		return
	end
	if value == candidate and value ~= ns.count then
		logChange(value)
		setCount(value)
	end
	candidate = value
end

local function onProbeUpdate()
	if (loaded(tensProbe) and loaded(onesProbe)) or GetTime() - waitStart > 2 then
		finishProbe()
	end
end

local function readBridge()
	if waiting then return end
	for _ = 1, TRIES do
		local n = (ns.db.bridgeKey or 0) % KEYS
		ns.db.bridgeKey = n + 1
		local wrapH, wrapV, filter
		slot, wrapH, wrapV, filter = keyAt(n)
		local name = ("s%03d"):format(slot)
		tensProbe:SetTexture(BRIDGE .. name .. "t.tga", wrapH, wrapV, filter)
		onesProbe:SetTexture(BRIDGE .. name .. "o.tga", wrapH, wrapV, filter)
		local cached = isLoaded(tensProbe) or isLoaded(onesProbe)
		if not cached then sawFreshLoad = true end
		if cached and sawFreshLoad then
			-- An old copy: skip it, it may hold a count from minutes ago.
			tensProbe:SetTexture(nil)
			onesProbe:SetTexture(nil)
			ns.cachedSkips = ns.cachedSkips + 1
		else
			ns.lastFresh = GetTime()
			waiting = true
			waitStart = GetTime()
			probeFrame:SetScript("OnUpdate", onProbeUpdate)
			return
		end
	end
	-- Every key we tried was cached. With ~13,800 keys this takes hours of
	-- play; a /reload clears the cache.
	if not ns.warnedStuck and ns.lastFresh and GetTime() - ns.lastFresh > 60 then
		ns.warnedStuck = true
		ns.Print("can't get a fresh reading from the companion right now. Type /reload to fix it.")
	end
end

ns.OnLogin(function()
	ref:SetTexture(ns.MEDIA .. "Icon")
	C_Timer.NewTicker(math.max(0.5, ns.db.interval or 1), readBridge)
	readBridge()
end)

---------------------------------------------------------------------------
-- Test mode: preview the badge without the companion.
---------------------------------------------------------------------------
function ns.RunTest()
	local steps = { 1, 3, 12, 100, 0 }
	local stepTime = 1.6
	testUntil = GetTime() + #steps * stepTime + 0.1
	for i, n in ipairs(steps) do
		C_Timer.After((i - 1) * stepTime, function() setCount(n, true) end)
	end
	C_Timer.After(#steps * stepTime + 0.2, function()
		-- The next real reading applies without a flash.
		ns.count = nil
		candidate = nil
	end)
end
