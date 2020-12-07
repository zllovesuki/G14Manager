package plugin

type Event int

const (
	EvtVolToggleMute Event = iota
	EvtKbReInit
	EvtKbBrightnessUp
	EvtKbBrightnessDown
	EvtKbBrightnessOff
	EvtKbToggleTouchpad
	EvtKbEmulateKeyPress
)

func (e Event) String() string {
	return [...]string{
		"Event: Toggling Mute",
		"Event: Keyboard Reinitialization",
		"Event: Keyboard Brightness Up",
		"Event: Keyboard Brightness Down",
		"Event: Keyboard Brightness Off",
		"Event: Keyboard Toggle Enable/Disable Touchpad",
		"Event: Keyboard Emulate KeyPress",
	}[e]
}
