package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"

	"github.com/zllovesuki/ROGManager/system/atkacpi"

	"github.com/zllovesuki/ROGManager/system/persist"
	"github.com/zllovesuki/ROGManager/system/thermal"
	"github.com/zllovesuki/ROGManager/util"

	"gopkg.in/toast.v1"
)

const (
	appName = "ROGManager"
)

type Controller interface {
	Run(haltCtx context.Context)
}

var _ Controller = &controller{}

type Config struct {
	Thermal  *thermal.Thermal
	Registry *persist.RegistryHelper
	ROGKey   []string
}

type keyedDebounce struct {
	noisy chan<- interface{}
	clean <-chan util.DebounceEvent
}

type controller struct {
	Config

	notifyQueueCh chan notification
	debounceCh    map[uint32]keyedDebounce
	keyCodeCh     chan uint32
}

func NewController(conf Config) (Controller, error) {
	if conf.Thermal == nil {
		return nil, errors.New("nil Thermal is invalid")
	}
	if conf.Registry == nil {
		return nil, errors.New("nil Registry is invalid")
	}
	if len(conf.ROGKey) == 0 {
		return nil, errors.New("empty key remap is invalid")
	}
	return &controller{
		Config:        conf,
		notifyQueueCh: make(chan notification, 10),
		debounceCh:    make(map[uint32]keyedDebounce),
		keyCodeCh:     make(chan uint32),
	}, nil
}

type notification struct {
	title   string
	message string
}

func (c *controller) notify(n notification) error {
	notification := toast.Notification{
		AppID:    appName,
		Title:    n.title,
		Message:  n.message,
		Duration: toast.Short,
		Audio:    "silent",
	}
	if err := notification.Push(); err != nil {
		return err
	}
	return nil
}

func (c *controller) initialize(haltCtx context.Context) {
	err := atkacpi.NewHidListener(haltCtx, c.keyCodeCh)
	if err != nil {
		log.Fatalln(err)
	}

	// TODO: revisit this
	keys := []uint32{
		58,  // ROG Key
		174, // Fn + F5
	}
	for _, key := range keys {
		// TODO: make debounce interval configurable, maybe
		in, out := util.Debounce(haltCtx, time.Millisecond*500)
		c.debounceCh[key] = keyedDebounce{
			noisy: in,
			clean: out,
		}
	}
}

func (c *controller) handleKeyPress(haltCtx context.Context) {
	for {
		select {
		case keyCode := <-c.keyCodeCh:
			switch keyCode {
			case 56:
				log.Println("ROG Key Pressed (debounced)")
				c.debounceCh[58].noisy <- struct{}{}
			case 174:
				log.Println("Fn + F5 Pressed (debounced)")
				c.debounceCh[174].noisy <- struct{}{}

			// TODO: Handle keyboard brightness up and down
			/*
				   case 87:
					   // On battery
				   case 107:
					   // Fn + F10: disable/enable trackpad
				   case 123:
					   // Power input change (unplug/plug in)
				   case 124:
					   // microphone mute/unmute
				   case 196:
					   // brightness up
				   case 197:
					   // brightness down
			*/
			default:
				log.Printf("Unknown keypress: %d\n", keyCode)
			}
		case <-haltCtx.Done():
			log.Println("Exiting handleKeyPress")
			return
		}
	}
}

func (c *controller) handleNotify(haltCtx context.Context) {
	for {
		select {
		case msg := <-c.notifyQueueCh:
			log.Println("Sending toast notification")
			if err := c.notify(msg); err != nil {
				log.Printf("Error sending toast notification: %s\n", err)
			}
		case <-haltCtx.Done():
			log.Println("Exiting handleNotify")
			return
		}
	}
}

func (c *controller) handleDebounce(haltCtx context.Context) {
	for {
		select {
		case ev := <-c.debounceCh[58].clean:
			log.Printf("ROG Key pressed %d times\n", ev.Counter)
			if int(ev.Counter) <= len(c.Config.ROGKey) {
				cmd := exec.Command("cmd.exe", "/C", c.Config.ROGKey[ev.Counter-1])
				cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
				if err := cmd.Start(); err != nil {
					log.Println(err)
				}
			}
		case ev := <-c.debounceCh[174].clean:
			log.Printf("Fn + F5 pressed %d times\n", ev.Counter)
			next, err := c.Config.Thermal.NextProfile(int(ev.Counter))
			message := fmt.Sprintf("Thermal plan changed to %s", next)
			if err != nil {
				log.Println(err)
				message = err.Error()
			}
			c.notifyQueueCh <- notification{
				title:   "Toggle Thermal Plan",
				message: message,
			}
			if err := c.Config.Registry.Save(); err != nil {
				log.Println("error saving to registry", err)
			}
		case <-haltCtx.Done():
			log.Println("Exiting handleDebounce")
			return
		}
	}
}

func (c *controller) Run(haltCtx context.Context) {
	// TODO: revisit this
	c.notifyQueueCh <- notification{
		title:   "Settings Loaded from Registry",
		message: fmt.Sprintf("Current Thermal Plan: %s", c.Config.Thermal.CurrentProfile().Name),
	}

	c.initialize(haltCtx)

	go c.handleNotify(haltCtx)
	go c.handleDebounce(haltCtx)
	go c.handleKeyPress(haltCtx)

	<-haltCtx.Done()
}
