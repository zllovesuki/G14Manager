package volume

import (
	"log"
	"sync"

	"github.com/moutend/go-wca/pkg/wca"
	"github.com/scjalliance/comshim"
)

// TODO: document this ole mess

type Control struct {
	sync.Mutex
	deferFuncs []func()
	dryRun     bool
	isMuted    bool
	// defaultOutput *wca.IMMDevice
	defaultInput *wca.IMMDevice
}

func NewControl(dryRun bool) (*Control, error) {
	c := &Control{
		dryRun: dryRun,
	}

	if err := c.checkMicrophoneMute(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Control) connect() (done func(), err error) {
	c.deferFuncs = make([]func(), 0, 1)

	comshim.Add(1)
	defer func() {
		if err != nil {
			comshim.Done()
		}
	}()
	c.deferFuncs = append(c.deferFuncs, comshim.Done)

	var mmde *wca.IMMDeviceEnumerator
	if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return
	}
	c.deferFuncs = append(c.deferFuncs, func() {
		mmde.Release()
	})

	if err = mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &c.defaultInput); err != nil {
		return
	}
	c.deferFuncs = append(c.deferFuncs, func() {
		c.defaultInput.Release()
	})

	/*if err := mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &c.defaultOutput); err != nil {
		return nil, err
	}
	c.deferFuncs = append(c.deferFuncs, func() {
		c.defaultOutput.Release()
	})*/

	done = func() {
		for i := len(c.deferFuncs) - 1; i >= 0; i-- {
			c.deferFuncs[i]()
		}
		c.deferFuncs = nil
	}
	return
}

func (c *Control) checkMicrophoneMute() (err error) {
	c.Lock()
	defer c.Unlock()

	done, err := c.connect()
	if err != nil {
		return
	}
	defer done()

	var ps *wca.IPropertyStore
	if err = c.defaultInput.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err = ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err != nil {
		return
	}
	log.Printf("wca: default microphone: %s\n", pv.String())

	var aev *wca.IAudioEndpointVolume
	if err = c.defaultInput.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return
	}
	defer aev.Release()

	if err = aev.GetMute(&c.isMuted); err != nil {
		return
	}

	log.Printf("wca: current microphone mute is %v\n", c.isMuted)

	return
}

func (c *Control) ToggleMicrophoneMute() error {
	c.Lock()
	defer c.Unlock()

	if c.dryRun {
		return nil
	}

	done, err := c.connect()
	if err != nil {
		return err
	}
	defer done()

	log.Printf("wca: setting microphone mute to %v\n", !c.isMuted)
	var aev *wca.IAudioEndpointVolume
	if err := c.defaultInput.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return err
	}
	defer aev.Release()

	if err := aev.SetMute(!c.isMuted, nil); err != nil {
		return err
	}

	c.isMuted = !c.isMuted

	return nil
}
