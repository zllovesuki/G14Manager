package atkacpi

import (
	"context"
	"fmt"

	"github.com/bi-zone/wmi"
)

type wmiEvent struct {
	Active       bool
	EventID      uint32
	InstanceName string
	TIME_CREATED uint64
}

// NewKeyListener will query WMI directly and return key code to the channel
func NewKeyListener(haltCtx context.Context, eventCh chan uint32) error {
	ch := make(chan wmiEvent)
	q, err := wmi.NewNotificationQuery(ch, `SELECT * FROM AsusAtkWmiEvent`)
	if err != nil {
		return fmt.Errorf("Failed to create NotificationQuery; %s", err)
	}
	q.SetConnectServerArgs(nil, `root\wmi`)

	go func() {
		q.StartNotifications()
	}()

	go func() {
		for {
			select {
			case ev := <-ch:
				eventCh <- ev.EventID
			case <-haltCtx.Done():
				q.Stop()
				return
			}
		}
	}()

	return nil
}
