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
	EvtSentinelCycleThermalProfile
	EvtSentinelUtilityKey
	EvtSentinelRestartGPU

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
		"Event (sentinel): Cycle thermal profile",
		"Event (sentinel): ROG/Utility Key",
		"Event (sentinel): Restart GPU",

		"Callback: Request to persist config",
		"Callback: Request to notify user",
	}[e]
}
