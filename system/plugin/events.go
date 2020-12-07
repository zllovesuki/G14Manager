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
	EvtSentinelCycleThermalProfile

	CbPersistConfig
	CbNotifyToast
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
		"Event (sentinel): Cycle thermal profile",

		"Callback: Request to persist config",
		"Callback: Request to notify user",
	}[e]
}
