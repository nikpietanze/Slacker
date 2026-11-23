-- Options under Esc > Options > AddOns > SlackNotify, built from Blizzard's
-- native settings controls so it is searchable and has a Defaults button. The
-- size options are also offered in the icon's Edit Mode dialog.
local _, ns = ...

local category, layout
local settings = {}
local fromEditMode

local function apply()
	if ns.ApplySettings then ns.ApplySettings() end
	-- Rebuilding the dialog mid-drag would interrupt its slider.
	if not fromEditMode and ns.RefreshEditModeDialog then ns.RefreshEditModeDialog() end
end

local function register(key, varType, name, default, onChange)
	local setting = Settings.RegisterAddOnSetting(category, "SlackNotify_" .. key, key, ns.db, varType, name, default)
	if onChange then
		Settings.SetOnValueChangedCallback("SlackNotify_" .. key, onChange)
	end
	settings[key] = setting
	return setting
end

-- Changes an option through its registered setting, so the options panel and
-- the Edit Mode dialog always agree and change callbacks run once.
function ns.SetOption(key, value)
	fromEditMode = true
	local setting = settings[key]
	if setting and setting.SetValue then
		setting:SetValue(value)
	else
		ns.db[key] = value
		apply()
	end
	fromEditMode = false
end

local function checkbox(key, name, tooltip, onChange, default)
	if default == nil then default = true end
	local setting = register(key, Settings.VarType.Boolean, name, default, onChange)
	return Settings.CreateCheckbox(category, setting, tooltip)
end

local function dropdown(key, name, default, choices, tooltip, onChange)
	local setting = register(key, Settings.VarType.String, name, default, onChange)
	return Settings.CreateDropdown(category, setting, function()
		local container = Settings.CreateControlTextContainer()
		for _, choice in ipairs(type(choices) == "function" and choices() or choices) do
			container:Add(choice.value, choice.text, choice.tooltip)
		end
		return container:GetData()
	end, tooltip)
end

local function slider(key, name, default, min, max, step, format, tooltip, onChange)
	local setting = register(key, Settings.VarType.Number, name, default, onChange)
	local options = Settings.CreateSliderOptions(min, max, step)
	options:SetLabelFormatter(MinimalSliderWithSteppersMixin.Label.Right, function(value)
		return format:format(value)
	end)
	return Settings.CreateSlider(category, setting, options, tooltip)
end

local function header(text)
	layout:AddInitializer(CreateSettingsListSectionHeaderInitializer(text))
end

local function button(name, text, onClick, tooltip)
	layout:AddInitializer(CreateSettingsButtonInitializer(name, text, onClick, tooltip, true))
end

local function soundChoices()
	local list = { { value = "none", text = "None" } }
	for _, sound in ipairs(ns.SOUNDS) do
		list[#list + 1] = { value = sound.key, text = sound.label }
	end
	local media = ns.SharedMedia()
	if media then
		for _, name in ipairs(media:List("sound")) do
			if name ~= "None" then
				list[#list + 1] = { value = "lsm:" .. name, text = name }
			end
		end
	end
	return list
end

ns.OnLoad(function()
	category, layout = Settings.RegisterVerticalLayoutCategory("SlackNotify")
	layout = layout or SettingsPanel:GetLayout(category)

	header("Icon")

	checkbox("locked", "Lock position",
		"When unlocked, drag the icon to move it. You can always move it in Edit Mode.", apply, false)
	checkbox("tooltip", "Tooltip", "Show how many notifications you have when you hover the icon.")
	slider("iconSize", "Icon size", 32, 20, 64, 2, "%dpx", "Size of the icon.", apply)
	slider("badgeScale", "Badge size", 100, 50, 200, 10, "%d%%", "Size of the red count badge.", apply)
	slider("opacity", "Opacity", 100, 20, 100, 5, "%d%%", "How solid the icon looks.", apply)

	button("Move icon", "Move", function() ns.Move() end,
		"Open Edit Mode and drag the icon where you want it. Select it there for size options.")
	button("Reset position", "Reset", function() ns.ResetPosition() end,
		"Move the icon back above the center of the screen.")

	header("New notifications")

	checkbox("flash", "Flash", "Flash the icon briefly when a new Slack notification arrives.")
	dropdown("sound", "Sound", "none", soundChoices,
		"Play a sound when a new Slack notification arrives. Choosing a sound plays it.", function()
			ns.PlaySound(ns.db.sound)
		end)
	button("Test badge", "Test", function() ns.RunTest() end,
		"Show sample counts on the icon for a few seconds, without the companion.")

	Settings.RegisterAddOnCategory(category)
end)

-- Settings shown when the icon is selected in Edit Mode.
function ns.EditModeSettings(lib)
	local types = lib.SettingType
	local function value(key)
		return function() return ns.db[key] end
	end
	local function setter(key)
		return function(_, v) ns.SetOption(key, v) end
	end
	local function pixels(v) return ("%dpx"):format(v) end
	local function percent(v) return ("%d%%"):format(v) end
	return {
		{ kind = types.Slider, name = "Icon size", default = 32, get = value("iconSize"), set = setter("iconSize"),
			minValue = 20, maxValue = 64, valueStep = 2, formatter = pixels },
		{ kind = types.Slider, name = "Badge size", default = 100, get = value("badgeScale"),
			set = setter("badgeScale"), minValue = 50, maxValue = 200, valueStep = 10, formatter = percent },
		{ kind = types.Slider, name = "Opacity", default = 100, get = value("opacity"), set = setter("opacity"),
			minValue = 20, maxValue = 100, valueStep = 5, formatter = percent },
	}
end

function ns.OpenOptions()
	if category then
		Settings.OpenToCategory(category:GetID())
	end
end
