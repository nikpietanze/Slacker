-- Reads the notification count from the companion app.
--
-- Addons can't read files, but they can load a texture and ask for its native
-- size. The companion writes Bridge\count.tga with a width and height that
-- encode the count, and we reload it about once a second.

local ADDON_NAME, ns = ...

local BRIDGE_PATH = [[Interface\AddOns\]] .. ADDON_NAME .. [[\Bridge\count.tga]]

---------------------------------------------------------------------------
-- Codec. Mirror of companion/internal/bridge/codec.go: code n maps to
-- exponents (a, b), enumerated by diagonals a+b = 0, 1, 2, ..., each <= 10,
-- and the texture is 2^(a+1) x 2^(b+1) px.
--   code 0       companion offline (the 2x2 placeholder shipped with the addon)
--   code 1..100  count 0..99
--   code 101     100 or more ("99+")
--
-- The file is 32-bit: an 8-bit grayscale bridge measured 1x1 on a real client
-- (1.60.1, macOS), while 32-bit files measure exactly. As a safeguard against
-- clients or UI scales that report sizes in other units, every read also
-- measures a texture of known size (the 64x64 icon), scales by that, and
-- rounds to the nearest power of two.
---------------------------------------------------------------------------
local MAX_EXP = 10
local NUM_CODES = ns.MAX_COUNT + 3
local REF_SIZE = 64 -- Media\Icon.tga
local LOG2 = math.log(2)
local codeByExp = {}
do
	local n, s = 0, 0
	while n < NUM_CODES do
		for a = 0, s do
			local b = s - a
			if n < NUM_CODES and a <= MAX_EXP and b <= MAX_EXP then
				codeByExp[a .. "," .. b] = n
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

-- Decodes a measured size. scaleW/scaleH convert UI units to file pixels
-- (measured reference size / real reference size); 1 when omitted.
-- Returns the count (0..100, 100 meaning 99+), false when the companion is
-- offline, or nil when the size isn't a bridge value (e.g. not loaded yet).
function ns.DecodeSize(w, h, scaleW, scaleH)
	if not w or not h or w <= 0 or h <= 0 then return nil end
	local a, b = exponent(w / (scaleW or 1)), exponent(h / (scaleH or 1))
	if not a or not b then return nil end
	local code = codeByExp[a .. "," .. b]
	if code == nil then return nil end
	if code == 0 then return false end
	return code - 1
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
-- Poller. The probe texture has no size of its own and a single anchor, so
-- once loaded, GetSize() reports the file's native pixel size. Clearing it
-- first lets the client drop its cached copy and read the file again.
---------------------------------------------------------------------------
local probeFrame = CreateFrame("Frame", nil, UIParent)
probeFrame:SetSize(1, 1)
probeFrame:SetPoint("BOTTOMLEFT")
local probe = probeFrame:CreateTexture(nil, "BACKGROUND")
probe:SetPoint("TOPLEFT")
probe:SetAlpha(0)
-- Same setup with a texture of known size, to measure the screen scaling.
local ref = probeFrame:CreateTexture(nil, "BACKGROUND")
ref:SetPoint("TOPLEFT")
ref:SetAlpha(0)

local waiting, waitStart = false, 0

local function finishProbe()
	waiting = false
	probeFrame:SetScript("OnUpdate", nil)
	local w, h = probe:GetSize()
	local rw, rh = ref:GetSize()
	ns.lastSize = ("%.2fx%.2f"):format(w, h)
	ns.lastRef = ("%.2fx%.2f"):format(rw, rh)
	ns.lastRead = GetTime()
	local value
	if rw > 0 and rh > 0 then
		value = ns.DecodeSize(w, h, rw / REF_SIZE, rh / REF_SIZE)
	end
	if ns.debug then
		ns.Print(("bridge %s, reference %s -> %s"):format(ns.lastSize, ns.lastRef, tostring(value)))
	end
	if value ~= nil then setCount(value) end
end

local function onProbeUpdate()
	local loaded = not probe.IsObjectLoaded or probe:IsObjectLoaded()
	local w = probe:GetSize()
	if (loaded and w and w > 0) or GetTime() - waitStart > 2 then
		finishProbe()
	end
end

local function readBridge()
	if waiting then return end
	waiting = true
	probe:SetTexture(nil)
	C_Timer.After(0.15, function()
		probe:SetTexture(BRIDGE_PATH)
		waitStart = GetTime()
		probeFrame:SetScript("OnUpdate", onProbeUpdate)
	end)
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
		ns.count = nil -- the next real reading applies without a flash
		readBridge()
	end)
end
