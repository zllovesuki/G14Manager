package volume

import (
	"context"
	"log"

	"github.com/moutend/go-wca/pkg/wca"
	"github.com/scjalliance/comshim"
)

// TODO: document this ole mess

type Control struct {
	dryRun  bool
	isMuted bool
	queue   chan task
}

type callbackFn func(defaultInput *wca.IMMDevice) (err error)

type task struct {
	fn  callbackFn
	err chan error
}

func NewVolumeControl(dryRun bool) (*Control, error) {
	c := &Control{
		dryRun: dryRun,
		queue:  make(chan task),
	}

	return c, nil
}

func (c *Control) Run(haltCtx context.Context) {
	log.Println("volCtrl: Starting queue loop")

	for {
		select {
		case t := <-c.queue:
			t.err <- c.inputDevice(t.fn)
		case <-haltCtx.Done():
			return
		}
	}
}

func (c *Control) inputDevice(fn callbackFn) (err error) {
	comshim.Add(1)
	defer comshim.Done()

	var mmde *wca.IMMDeviceEnumerator
	if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return
	}
	defer mmde.Release()

	var defaultInput *wca.IMMDevice
	if err = mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &defaultInput); err != nil {
		return
	}
	defer defaultInput.Release()

	err = fn(defaultInput)

	return
}

func (c *Control) checkMuteFn(defaultInput *wca.IMMDevice) (err error) {
	var ps *wca.IPropertyStore
	if err = defaultInput.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err = ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err != nil {
		return
	}
	log.Printf("wca: default microphone: %s\n", pv.String())

	var aev *wca.IAudioEndpointVolume
	if err = defaultInput.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return
	}
	defer aev.Release()

	if err = aev.GetMute(&c.isMuted); err != nil {
		return
	}

	log.Printf("wca: current microphone mute is %v\n", c.isMuted)

	return
}

func (c *Control) setMuteFn(defaultInput *wca.IMMDevice) (err error) {
	log.Printf("wca: setting microphone mute to %v\n", !c.isMuted)
	var aev *wca.IAudioEndpointVolume
	if err = defaultInput.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &aev); err != nil {
		return
	}
	defer aev.Release()

	if err = aev.SetMute(!c.isMuted, nil); err != nil {
		return
	}
	c.isMuted = !c.isMuted

	return
}

func (c *Control) CheckMicrophoneMute() error {
	return c.doWork(c.checkMuteFn)
}

func (c *Control) ToggleMicrophoneMute() error {
	if c.dryRun {
		return nil
	}

	return c.doWork(c.setMuteFn)
}

func (c *Control) doWork(fn callbackFn) error {
	err := make(chan error)
	c.queue <- task{
		fn:  fn,
		err: err,
	}
	return <-err
}
