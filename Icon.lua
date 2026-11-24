-- The icon: a small speech bubble with an "S" (baked into Media\Icon.tga) and
-- a red count badge. Positioned through Blizzard's Edit Mode when LibEditMode
-- is present, or by dragging while unlocked. Plain (non-secure) frames only,
-- so nothing here is ever blocked or tainted in combat.

local _, ns = ...

local LEM = LibStub and LibStub("LibEditMode", true)
local BADGE_HEIGHT = 16 -- At a 32px icon; scales with icon size and badge size.

local button, icon, glow, glowAnim, halo, haloAnim, badgePulse
local ping, pingAnim, popAnim
local ALERT_ROUNDS = 3
local badge, badgeText, badgeLeft, badgeMid, badgeRight
local layoutKey = "default"
local editing = false

-- Digits use the client's number font (Arial Narrow on Western clients).
local function numberFont()
	return (NumberFontNormal and NumberFontNormal:GetFont())
		or (GameFontNormal and GameFontNormal:GetFont())
		or STANDARD_TEXT_FONT
end

---------------------------------------------------------------------------
-- Position, saved per Edit Mode layout
---------------------------------------------------------------------------
function ns.LayoutKey() return layoutKey end

local function position()
	return ns.db.positions[layoutKey] or ns.db.positions.default or ns.DEFAULT_POSITION
end

local function applyPosition()
	local p = position()
	button:ClearAllPoints()
	button:SetPoint(p.point, UIParent, p.point, p.x, p.y)
end

local function savePosition(point, x, y, key)
	ns.db.positions[key or layoutKey] = { point = point, x = x, y = y }
end

function ns.ResetPosition()
	ns.db.positions[layoutKey] = CopyTable(ns.DEFAULT_POSITION)
	applyPosition()
end

local function editModeAvailable()
	return LEM and EditModeManagerFrame and ShowUIPanel and true or false
end
ns.EditModeAvailable = editModeAvailable

-- Opens Edit Mode when available; otherwise unlocks the icon for dragging.
function ns.Move()
	if SettingsPanel and SettingsPanel:IsShown() then HideUIPanel(SettingsPanel) end
	if editModeAvailable() and not InCombatLockdown() then
		ShowUIPanel(EditModeManagerFrame)
	else
		ns.SetOption("locked", false)
		ns.Print("Drag the icon to move it, then lock it with /slacker lock.")
	end
end

---------------------------------------------------------------------------
-- Badge
---------------------------------------------------------------------------
local function layoutBadge(text)
	badgeText:SetText(text)
	local h = BADGE_HEIGHT
	local w = h
	if #text > 1 then
		w = math.max(h, math.ceil(badgeText:GetStringWidth()) + 9)
	end
	badge:SetSize(w, h)
	badgeLeft:SetWidth(h / 2)
	badgeRight:SetWidth(h / 2)
	badgeMid:SetShown(w > h)
end

local function createBadge()
	badge = CreateFrame("Frame", nil, button)
	badge:SetFrameLevel(button:GetFrameLevel() + 3)

	local tex = ns.MEDIA .. "Badge"
	badgeLeft = badge:CreateTexture(nil, "ARTWORK")
	badgeLeft:SetTexture(tex)
	badgeLeft:SetTexCoord(0, 0.5, 0, 1)
	badgeLeft:SetPoint("TOPLEFT")
	badgeLeft:SetPoint("BOTTOMLEFT")

	badgeRight = badge:CreateTexture(nil, "ARTWORK")
	badgeRight:SetTexture(tex)
	badgeRight:SetTexCoord(0.5, 1, 0, 1)
	badgeRight:SetPoint("TOPRIGHT")
	badgeRight:SetPoint("BOTTOMRIGHT")

	-- The middle column of the disc, stretched: turns it into a pill for "12" or "99+".
	badgeMid = badge:CreateTexture(nil, "ARTWORK")
	badgeMid:SetTexture(tex)
	badgeMid:SetTexCoord(0.49, 0.51, 0, 1)
	badgeMid:SetPoint("TOPLEFT", badgeLeft, "TOPRIGHT")
	badgeMid:SetPoint("BOTTOMRIGHT", badgeRight, "BOTTOMLEFT")

	badgeText = badge:CreateFontString(nil, "OVERLAY")
	badgeText:SetFont(numberFont(), 11, "")
	badgeText:SetTextColor(1, 1, 1)
	badgeText:SetShadowColor(0, 0, 0, 0.6)
	badgeText:SetShadowOffset(0.5, -0.5)
	badgeText:SetPoint("CENTER", badge, "CENTER", 0.5, 0)

	badge:Hide()
end

---------------------------------------------------------------------------
-- Tooltip
---------------------------------------------------------------------------
local function showTooltip(self)
	if not ns.db.tooltip or editing then return end
	local _, y = self:GetCenter()
	local top = y and y > UIParent:GetHeight() / 2
	GameTooltip:SetOwner(self, top and "ANCHOR_BOTTOMLEFT" or "ANCHOR_TOPLEFT")
	GameTooltip:AddLine("Slack: " .. ns.Describe(ns.count), 1, 1, 1)
	if ns.count == false then
		GameTooltip:AddLine("Slacker needs its free companion app to see Slack.", 0.7, 0.7, 0.7, true)
		GameTooltip:AddLine("Click to get it, right-click for options", 0.5, 0.5, 0.5)
		GameTooltip:Show()
		return
	elseif type(ns.count) == "number" and ns.count > 0 and ns.lastNotified then
		GameTooltip:AddLine("Last: " .. date("%H:%M", ns.lastNotified), 0.7, 0.7, 0.7)
	end
	GameTooltip:AddLine(ns.db.locked and "Right-click for options" or "Drag to move, right-click for options",
		0.5, 0.5, 0.5)
	GameTooltip:Show()
end

---------------------------------------------------------------------------
-- Refresh and settings
---------------------------------------------------------------------------
function ns.Refresh()
	if not button then return end
	local count = ns.count
	-- Edit Mode shows a sample badge so you can size it.
	if editing then count = 3 end
	local active = type(count) == "number" and count > 0
	local opacity = ns.db.opacity / 100
	-- A slow breathing glow while anything is unread, so it's hard to miss.
	if active and ns.db.pulse and not editing then
		if not haloAnim:IsPlaying() then
			haloAnim:Play()
			badgePulse:Play()
		end
	elseif haloAnim:IsPlaying() then
		haloAnim:Stop()
		badgePulse:Stop()
	end
	icon:SetDesaturated(not active)
	if active then
		button:SetAlpha(opacity)
		layoutBadge(count > ns.MAX_COUNT and "99+" or tostring(count))
		badge:Show()
	else
		-- Muted when clear, dimmer still when the companion isn't running.
		button:SetAlpha(opacity * (count == false and 0.45 or 0.7))
		badge:Hide()
	end
	if GameTooltip:IsOwned(button) then showTooltip(button) end
end

function ns.ApplySettings()
	if not button then return end
	local size = ns.db.iconSize
	button:SetSize(size, size)
	glow:SetSize(size * 2.4, size * 2.4)
	ping:SetSize(size * 1.1, size * 1.1)
	halo:SetSize(size * 2, size * 2)
	-- The badge is designed for a 32px icon.
	badge:SetScale(size / 32 * ns.db.badgeScale / 100)
	badge:ClearAllPoints()
	badge:SetPoint("CENTER", button, "TOPRIGHT", -4 * size / 32, -5 * size / 32)
	ns.Refresh()
end

-- One round of the new-notification alert: a bright flash, a ripple out
-- from the icon, and a quick bounce. It runs ALERT_ROUNDS times, much louder
-- than the slow pulse that keeps going until the notifications are read.
local function alertRound()
	glowAnim:Stop()
	pingAnim:Stop()
	popAnim:Stop()
	popAnim.loops = 0
	glowAnim:Play()
	pingAnim:Play()
	popAnim:Play()
end

function ns.OnNewNotification()
	if ns.db.flash and glowAnim then
		pingAnim.rounds = 1
		alertRound()
	end
	ns.PlaySound(ns.db.sound)
end

---------------------------------------------------------------------------
-- Edit Mode
---------------------------------------------------------------------------
local function setupEditMode()
	if not LEM then return end
	LEM:AddFrame(button, function(_, layoutName, point, x, y)
		savePosition(point, x, y, layoutName or "default")
	end, ns.DEFAULT_POSITION, "Slacker")
	LEM:RegisterCallback("enter", function()
		editing = true
		GameTooltip:Hide()
		ns.Refresh()
	end)
	LEM:RegisterCallback("exit", function()
		editing = false
		ns.Refresh()
	end)
	LEM:RegisterCallback("layout", function(layoutName)
		layoutKey = layoutName or "default"
		-- A layout without its own position starts from the last one used.
		if not ns.db.positions[layoutKey] and ns.db.positions.default then
			ns.db.positions[layoutKey] = CopyTable(ns.db.positions.default)
		end
		applyPosition()
	end)
	if ns.EditModeSettings then
		LEM:AddFrameSettings(button, ns.EditModeSettings(LEM))
	end
	if LEM.AddFrameSettingsButton then
		LEM:AddFrameSettingsButton(button, {
			text = "More options",
			click = function()
				HideUIPanel(EditModeManagerFrame)
				ns.OpenOptions()
			end,
		})
	end
end

function ns.RefreshEditModeDialog()
	if LEM and LEM.RefreshFrameSettings and LEM:IsInEditMode() then
		LEM:RefreshFrameSettings(button)
	end
end

---------------------------------------------------------------------------
-- Construction
---------------------------------------------------------------------------
ns.OnLogin(function()
	button = CreateFrame("Button", "SlackerButton", UIParent)
	button:SetFrameStrata("MEDIUM")
	button:SetClampedToScreen(true)
	button:SetMovable(true)
	button:EnableMouse(true)
	button:RegisterForDrag("LeftButton")
	button:RegisterForClicks("LeftButtonUp", "RightButtonUp")
	if button.SetDontSavePosition then button:SetDontSavePosition(true) end

	glow = button:CreateTexture(nil, "BACKGROUND")
	glow:SetTexture(ns.MEDIA .. "Glow")
	glow:SetBlendMode("ADD")
	glow:SetVertexColor(1, 0.35, 0.35)
	glow:SetPoint("CENTER", 0, 1)
	glow:SetAlpha(0)

	glowAnim = glow:CreateAnimationGroup()
	local up = glowAnim:CreateAnimation("Alpha")
	up:SetFromAlpha(0)
	up:SetToAlpha(1)
	up:SetDuration(0.15)
	up:SetOrder(1)
	local down = glowAnim:CreateAnimation("Alpha")
	down:SetFromAlpha(1)
	down:SetToAlpha(0)
	down:SetDuration(0.6)
	down:SetOrder(2)

	-- A ring that ripples outward from the icon and fades.
	ping = button:CreateTexture(nil, "BACKGROUND", nil, 1)
	ping:SetTexture(ns.MEDIA .. "Ring")
	-- Normal blending so it shows on bright scenery too.
	ping:SetBlendMode("BLEND")
	ping:SetVertexColor(1, 0.25, 0.27)
	ping:SetPoint("CENTER", 0, 1)
	ping:SetAlpha(0)

	pingAnim = ping:CreateAnimationGroup()
	local grow = pingAnim:CreateAnimation("Scale")
	if grow.SetScaleFrom then
		grow:SetScaleFrom(1, 1)
		grow:SetScaleTo(2.6, 2.6)
	else
		grow:SetFromScale(1, 1)
		grow:SetToScale(2.6, 2.6)
	end
	grow:SetOrigin("CENTER", 0, 0)
	grow:SetDuration(0.75)
	if grow.SetSmoothing then grow:SetSmoothing("OUT") end
	local fade = pingAnim:CreateAnimation("Alpha")
	fade:SetFromAlpha(1)
	fade:SetToAlpha(0)
	fade:SetDuration(0.75)
	if fade.SetSmoothing then fade:SetSmoothing("IN") end
	pingAnim:SetScript("OnFinished", function(self)
		if (self.rounds or ALERT_ROUNDS) < ALERT_ROUNDS then
			self.rounds = self.rounds + 1
			alertRound()
		else
			popAnim:Stop()
		end
	end)

	halo = button:CreateTexture(nil, "BACKGROUND", nil, -1)
	halo:SetTexture(ns.MEDIA .. "Glow")
	-- Normal blending (not additive) so it shows on bright scenery too.
	halo:SetBlendMode("BLEND")
	halo:SetVertexColor(1, 0.2, 0.22)
	halo:SetPoint("CENTER", 0, 1)
	halo:SetAlpha(0)

	haloAnim = halo:CreateAnimationGroup()
	haloAnim:SetLooping("BOUNCE")
	local breathe = haloAnim:CreateAnimation("Alpha")
	breathe:SetFromAlpha(0.1)
	breathe:SetToAlpha(0.9)
	breathe:SetDuration(0.7)
	if breathe.SetSmoothing then breathe:SetSmoothing("IN_OUT") end

	icon = button:CreateTexture(nil, "ARTWORK")
	icon:SetTexture(ns.MEDIA .. "Icon")
	icon:SetAllPoints()

	-- The bubble jumps out and back once per alert round.
	popAnim = icon:CreateAnimationGroup()
	popAnim:SetLooping("BOUNCE")
	local pop = popAnim:CreateAnimation("Scale")
	if pop.SetScaleFrom then
		pop:SetScaleFrom(1, 1)
		pop:SetScaleTo(1.3, 1.3)
	else
		pop:SetFromScale(1, 1)
		pop:SetToScale(1.3, 1.3)
	end
	pop:SetOrigin("CENTER", 0, 0)
	pop:SetDuration(0.15)
	if pop.SetSmoothing then pop:SetSmoothing("OUT") end
	-- Out, then back: stop after the second half so it ends at full size.
	popAnim:SetScript("OnLoop", function(self)
		self.loops = (self.loops or 0) + 1
		if self.loops >= 2 then self:Stop() end
	end)

	createBadge()

	-- The badge swells gently in time with the halo.
	badgePulse = badge:CreateAnimationGroup()
	badgePulse:SetLooping("BOUNCE")
	local swell = badgePulse:CreateAnimation("Scale")
	if swell.SetScaleFrom then
		swell:SetScaleFrom(1, 1)
		swell:SetScaleTo(1.2, 1.2)
	else
		swell:SetFromScale(1, 1)
		swell:SetToScale(1.2, 1.2)
	end
	swell:SetOrigin("CENTER", 0, 0)
	swell:SetDuration(0.7)
	if swell.SetSmoothing then swell:SetSmoothing("IN_OUT") end

	button:SetScript("OnDragStart", function(self)
		if not ns.db.locked and not editing then self:StartMoving() end
	end)
	button:SetScript("OnDragStop", function(self)
		self:StopMovingOrSizing()
		if editing then return end
		if self.SetUserPlaced then self:SetUserPlaced(false) end
		-- Store as an offset from the screen centre, like Edit Mode does.
		local cx, cy = self:GetCenter()
		local px, py = UIParent:GetCenter()
		savePosition("CENTER", math.floor(cx - px + 0.5), math.floor(cy - py + 0.5))
		applyPosition()
	end)
	button:SetScript("OnClick", function(_, mouseButton)
		if editing then return end
		if mouseButton == "RightButton" then
			ns.OpenOptions()
		elseif ns.count == false then
			ns.ShowSetup()
		end
	end)
	button:SetScript("OnEnter", showTooltip)
	button:SetScript("OnLeave", function() GameTooltip:Hide() end)

	-- If Edit Mode support fails on this client, dragging still works.
	if LEM and not pcall(setupEditMode) then
		LEM = nil
	end
	applyPosition()
	ns.button = button
	ns.ui = {
		badge = badge, badgeText = badgeText, icon = icon,
		glowAnim = glowAnim, pingAnim = pingAnim, popAnim = popAnim, haloAnim = haloAnim,
	}
	ns.ApplySettings()
end)
