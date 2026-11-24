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
-- The client keeps a texture cached until a periodic cleanup frees unused
-- ones (irregularly, from seconds to over a minute apart on 1.60.1), so
-- loading the same file again usually returns the old size. A file that
-- hasn't been loaded since the last cleanup is always read from disk.
-- The companion writes the count into SLOTS identical slots and we read the
-- next one each second, so each slot rests for about two minutes between
-- reads: long enough that a cleanup has almost always freed it.
--
-- The probes hold a slot only for the frame or two it takes to measure it,
-- so it's unused (and can be freed) the rest of the time. A value counts
-- only once two reads in a row agree, which skips a slot caught mid-write.
---------------------------------------------------------------------------
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
local LOG_SIZE = 30

-- Keeps the last few changes (with times) in the saved settings, so a bug
-- report can show how quickly the client picked up each one.
local function logChange(value)
	local log = ns.db.bridgeLog or {}
	ns.db.bridgeLog = log
	log[#log + 1] = ("%s slot %d %s -> %s"):format(date("%H:%M:%S"), slot, ns.lastSize, tostring(value))
	while #log > LOG_SIZE do table.remove(log, 1) end
end

local function loaded(t)
	return (not t.IsObjectLoaded or t:IsObjectLoaded()) and (t:GetSize()) > 0
end

local function finishProbe()
	waiting = false
	probeFrame:SetScript("OnUpdate", nil)
	local tw, th = tensProbe:GetSize()
	local ow, oh = onesProbe:GetSize()
	-- Let go right away so the cleanup can free them.
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
	waiting = true
	slot = slot % (SLOTS - 1) + 1 -- EXPERIMENT: slot 128 is the test file
	local name = ("s%03d"):format(slot)
	tensProbe:SetTexture(BRIDGE .. name .. "t.tga")
	onesProbe:SetTexture(BRIDGE .. name .. "o.tga")
	waitStart = GetTime()
	probeFrame:SetScript("OnUpdate", onProbeUpdate)
end

---------------------------------------------------------------------------
-- EXPERIMENT 2 (not for release). The companion side flips s128o between
-- 2x2 and 2x4 every 5 s of wall-clock time, and also creates a brand-new file
-- Bridge\n<epoch>.tga (same value) every second. Each second we read:
--   A  s128o through the usual path (control: cached)
--   D  s128o through a path with ".\" segments, unique each read
--   S  s128o through a path with doubled "\\" separators, unique each read
--   N  the brand-new file from two seconds ago (are new files visible?)
--   M  a file that never exists
---------------------------------------------------------------------------
local EXP_SECONDS = 120
local function spread(n, sep)
	-- n -> four counts 0..3 (base 4), each a run of sep after one folder
	local parts = { "Interface", "AddOns", ADDON_NAME, "Bridge" }
	local out = {}
	for i, part in ipairs(parts) do
		local k = math.floor(n / 4 ^ (i - 1)) % 4
		out[#out + 1] = part .. "\\" .. string.rep(sep, k)
	end
	return table.concat(out)
end
local probes = {}
for _, k in ipairs({ "A", "D", "S", "N", "M" }) do probes[k] = probeTexture() end
local function expStart()
	local log = {}
	ns.db.cacheExp2 = log
	local n, stop, done = 0, GetTime() + EXP_SECONDS, false
	local frame = CreateFrame("Frame")
	C_Timer.NewTicker(1, function()
		if done then return end
		if GetTime() > stop then done = true; log[#log + 1] = "done"; return end
		n = n + 1
		local t0 = time()
		local paths = {
			A = BRIDGE .. "s128o.tga",
			D = spread(n, ".\\") .. "s128o.tga",
			S = spread(n, "\\") .. "s128o.tga",
			N = BRIDGE .. "n" .. (t0 - 2) .. ".tga",
			M = BRIDGE .. "missing" .. n .. ".tga",
		}
		local now = {}
		for k, path in pairs(paths) do
			local ok = probes[k]:SetTexture(path)
			now[k] = (probes[k]:IsObjectLoaded() and "C" or "-") .. (ok and "" or "!")
		end
		local frames = 0
		frame:SetScript("OnUpdate", function(self)
			frames = frames + 1
			if frames < 30 then return end -- half a second for everything to load
			self:SetScript("OnUpdate", nil)
			local line = { t0, math.floor(t0 / 5) % 2, math.floor((t0 - 2) / 5) % 2 }
			for _, k in ipairs({ "A", "D", "S", "N", "M" }) do
				local w, h = probes[k]:GetSize()
				line[#line + 1] = ("%s%s%.0fx%.0f"):format(k, now[k], w, h)
				probes[k]:SetTexture(nil)
			end
			if n <= 3 then line[#line + 1] = paths.D .. " " .. paths.S end
			log[#log + 1] = table.concat(line, " ")
		end)
	end)
	ns.Print("cache experiment 2 running for " .. EXP_SECONDS .. " s. /reload after it says done.")
	C_Timer.After(EXP_SECONDS + 2, function() ns.Print("cache experiment 2 done, please /reload") end)
end

ns.OnLogin(function()
	C_Timer.After(3, expStart)
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
