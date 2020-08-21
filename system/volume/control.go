package volume

import (
	"log"
	"sync"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// TODO: document this ole mess

type Control struct {
	sync.Mutex
	deferFuncs    []func()
	isMuted       bool
	defaultOutput *wca.IMMDevice
	defaultInput  *wca.IMMDevice
}

func NewControl() (*Control, error) {

	log.Println("wca: opening ole connection to Core Audio API")

	ctrl := &Control{}

	// Investigate how COINIT_APARTMENTTHREADED vs COINIT_MULTITHREADED will affect wmi listener
	// right now we have to use COINIT_APARTMENTTHREADED
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return nil, err
	}
	ctrl.deferFuncs = append(ctrl.deferFuncs, ole.CoUninitialize)

	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return nil, err
	}
	ctrl.deferFuncs = append(ctrl.deferFuncs, func() {
		mmde.Release()
	})

	if err := mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &ctrl.defaultInput); err != nil {
		return nil, err
	}
	ctrl.deferFuncs = append(ctrl.deferFuncs, func() {
		ctrl.defaultInput.Release()
	})

	if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &ctrl.defaultOutput); err != nil {
		return nil, err
	}
	ctrl.deferFuncs = append(ctrl.deferFuncs, func() {
		ctrl.defaultOutput.Release()
	})

	if err := ctrl.checkMicrophoneMute(); err != nil {
		return nil, err
	}

	return ctrl, nil
}

func (c *Control) checkMicrophoneMute() error {
	c.Lock()
	defer c.Unlock()

	var ps *wca.IPropertyStore
	if err := c.defaultInput.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return err
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err := ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err != nil {
		return err
	}
	log.Printf("wca: default microphone: %s\n", pv.String())

	var aev *wca.IAudioEndpointVolume
	if err := c.defaultInput.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return err
	}
	defer aev.Release()

	if err := aev.GetMute(&c.isMuted); err != nil {
		return err
	}

	log.Printf("wca: current microphone mute is %v\n", c.isMuted)

	return nil
}

func (c *Control) setMicrophoneMute(isMute bool) error {
	c.Lock()
	defer c.Unlock()

	var aev *wca.IAudioEndpointVolume
	if err := c.defaultInput.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return err
	}
	defer aev.Release()

	if err := aev.SetMute(isMute, nil); err != nil {
		return err
	}

	c.isMuted = isMute

	return nil
}

func (c *Control) ToggleMicrophoneMute() error {
	log.Printf("wca: setting microphone mute to %v\n", !c.isMuted)
	return c.setMicrophoneMute(!c.isMuted)
}

func (c *Control) Close() {
	c.Lock()
	defer c.Unlock()

	for i := len(c.deferFuncs) - 1; i >= 0; i-- {
		c.deferFuncs[i]()
	}
}
