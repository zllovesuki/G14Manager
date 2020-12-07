package plugin

// Event defines the type of notification from controller to plugins
type Event int

// Define all the possible controller->plugin notifications
const (
	EvtKeyboardFn Event = iota
	EvtACPISuspend
	EvtACPIResume
	EvtChargerPluggedIn
	EvtChargerUnplugged
	EvtSentinelInitKeyboard
	EvtSentinelKeyboardBrightnessOff
)

func (e Event) String() string {
	return [...]string{
		"Event: Keyboard hardware function",
		"Event: ACPI Suspend",
		"Event: ACPI Resume",
		"Event: Charged plugged in",
		"Event: Charged unplugged",
		"Event (sentinel): Initializa keyboard",
		"Event (sentinel): Keyboard backlight off",
	}[e]
}

// Callback defines the type of notification from plugins to controller
type Callback int

// Define all the possible plugin->controller callbacks
const (
	CbPersistConfig Callback = iota
)

func (c Callback) String() string {
	return [...]string{
		"Callback: Request to save config",
	}[c]
}
