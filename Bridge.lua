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
---------------------------------------------------------------------------
local MAX_EXP = 10
local NUM_CODES = ns.MAX_COUNT + 3
local codeBySize = {}
do
	local n, s = 0, 0
	while n < NUM_CODES do
		for a = 0, s do
			local b = s - a
			if n < NUM_CODES and a <= MAX_EXP and b <= MAX_EXP then
				codeBySize[("%dx%d"):format(2 ^ (a + 1), 2 ^ (b + 1))] = n
				n = n + 1
			end
		end
		s = s + 1
	end
end

-- Returns the count (0..100, 100 meaning 99+), false when the companion is
-- offline, or nil when the size isn't a bridge value (e.g. not loaded yet).
function ns.DecodeSize(w, h)
	if not w or not h then return nil end
	local code = codeBySize[("%dx%d"):format(math.floor(w + 0.5), math.floor(h + 0.5))]
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

local waiting, waitStart = false, 0

local function finishProbe()
	waiting = false
	probeFrame:SetScript("OnUpdate", nil)
	local w, h = probe:GetSize()
	ns.lastSize = ("%dx%d"):format(math.floor(w + 0.5), math.floor(h + 0.5))
	ns.lastRead = GetTime()
	local value = ns.DecodeSize(w, h)
	if ns.debug then
		ns.Print(("bridge %s -> %s"):format(ns.lastSize, tostring(value)))
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
