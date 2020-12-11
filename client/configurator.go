package client

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rivo/tview"
	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"google.golang.org/grpc"
)

type Configurator struct {
	conn         *grpc.ClientConn
	gConfigsList protocol.ConfigListClient
	gThermal     protocol.ThermalClient
	gBattery     protocol.BatteryChargeLimitClient
	gKeyboard    protocol.KeyboardBrightnessClient
	gManager     protocol.ManagerControlClient

	ctx      context.Context
	cancelFn context.CancelFunc

	app    *tview.Application
	layers *tview.Pages

	connectModal *tview.Modal

	confirmationModal *tview.Modal
	confirmYes        func()
	confirmNo         string

	container *tview.Flex

	configView *tview.TextView

	fnLists     *tview.List
	fnListItems []listItem
}

type listItem struct {
	Main      string
	Secondary string
	Shortcut  rune
	Callback  func()
}

func NewInterface() *Configurator {
	return &Configurator{
		app:               tview.NewApplication(),
		layers:            tview.NewPages(),
		connectModal:      tview.NewModal(),
		confirmationModal: tview.NewModal(),
		container:         tview.NewFlex(),
		configView:        tview.NewTextView(),
		fnLists:           tview.NewList(),
	}
}

func (i *Configurator) connect(haltCtx context.Context) error {
	ctx, cancel := context.WithTimeout(haltCtx, time.Second*1)
	defer cancel()
	c, err := grpc.DialContext(ctx, "127.0.0.1:9963", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	i.conn = c
	i.gConfigsList = protocol.NewConfigListClient(c)
	i.gThermal = protocol.NewThermalClient(c)
	i.gBattery = protocol.NewBatteryChargeLimitClient(c)
	i.gKeyboard = protocol.NewKeyboardBrightnessClient(c)
	i.gManager = protocol.NewManagerControlClient(c)

	return nil
}

func (i *Configurator) setup() {
	i.layers.
		AddPage("connect", i.connectModal, true, true).
		AddPage("container", i.container, true, false).
		AddPage("confirmation", i.confirmationModal, true, false)

	i.setupFnList()
	i.setupModals()

	i.setupStyles()
	i.keyBindings()

	i.container.
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(i.fnLists, 0, 8, true).
				AddItem(tview.NewBox().SetTitle(" Version ").SetBorder(true), 0, 2, false),
			0, 4, true).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(i.configView, 0, 2, false).
				AddItem(tview.NewBox().SetTitle(" Change Settings ").SetBorder(true), 0, 4, false),
			0, 6, false)

	i.configView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyESC {
			i.app.SetFocus(i.fnLists)
		}
	})

	i.app.SetRoot(i.layers, true)
}

func (i *Configurator) setupModals() {
	i.connectModal.SetText("Connect to G14Manager Supervisor").
		AddButtons([]string{"Connect", "Quit"}).
		SetBackgroundColor(tcell.Color104).
		SetDoneFunc(func(index int, label string) {
			if label == "Connect" {
				err := i.connect(i.ctx)
				if err != nil {
					return
				}
				i.layers.SwitchToPage("container")
			} else {
				i.cancelFn()
			}
		})

	i.confirmationModal.SetText("Are you sure?").
		AddButtons([]string{"Yes", "No"}).
		SetBackgroundColor(tcell.Color104).
		SetDoneFunc(func(index int, label string) {
			switch label {
			case "Yes":
				i.confirmYes()
			case "No":
				i.layers.SwitchToPage(i.confirmNo)
			}
		})
}

func (i *Configurator) setupFnList() {
	i.fnListItems = []listItem{
		{
			Main:      "Manage Controller",
			Secondary: "Start/Stop Controller",
			Shortcut:  'm',
			Callback:  i.selectManager,
		},
		{
			Main:      "Configs",
			Secondary: "Get/Set features and profiles",
			Shortcut:  'c',
			Callback:  i.selectConfigs,
		},
		{
			Main:      "Thermal Profile",
			Secondary: "Get/Set thermal profile",
			Shortcut:  't',
			Callback:  i.selectThermal,
		},
		{
			Main:      "Keyboarc Backlight",
			Secondary: "Get/Set backlight level",
			Shortcut:  'k',
			Callback:  i.selectKeyboard,
		},
		{
			Main:      "Battery Charge Limit",
			Secondary: "Get/Set charge limit",
			Shortcut:  'b',
			Callback:  i.selectBattery,
		},
		{
			Main:      "Exit",
			Secondary: "Exit the Configurator",
			Shortcut:  'q',
			Callback: func() {
				i.confirmNo = "container"
				i.confirmYes = func() {
					i.cancelFn()
				}
				i.layers.SwitchToPage("confirmation")
			},
		},
	}
	for _, item := range i.fnListItems {
		i.fnLists.AddItem(item.Main, item.Secondary, item.Shortcut, item.Callback)
	}
}

func (i *Configurator) keyBindings() {
	// Right key on function list will select the item
	i.fnLists.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRight {
			i.fnListItems[i.fnLists.GetCurrentItem()].Callback()
			return nil
		}
		return event
	})

	// Left key on configView will go back to function list
	i.configView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft {
			i.app.SetFocus(i.fnLists)
			return nil
		}
		return event
	})
}

func (i *Configurator) setupStyles() {
	i.container.Box.SetBorder(true).SetTitle(" G14Manager Configurator ").SetBorderColor(tcell.ColorDarkSlateGrey)
	i.fnLists.Box.SetBorder(true).SetTitle(" Functions ")
	i.configView.Box.SetBorder(true).SetBorderAttributes(tcell.AttrNone).SetTitle(" Current Settings ")
}

func (i *Configurator) selectManager() {
	auto, err := i.gManager.GetCurrentAutoStart(context.Background(), &empty.Empty{})
	if err != nil {
		i.configView.SetText(err.Error())
		return
	}
	m, err := i.gManager.GetCurrentState(context.Background(), &empty.Empty{})
	if err != nil {
		i.configView.SetText(err.Error())
	}
	var txt string
	if auto.Success != true {
		txt = fmt.Sprintf("%s\n\nCannot get AutoStart status: %s", txt, auto.Message)
	} else {
		txt = fmt.Sprintf("%s\n\nRun Controller when Supervisor starts: %t", txt, auto.AutoStart)
	}
	if m.Success != true {
		txt = fmt.Sprintf("%s\n\nCannot get Manager running state: %s", txt, m.Message)
	} else {
		txt = fmt.Sprintf("%s\n\nController is currently: %s", txt, m.State)
	}
	i.configView.SetText(txt)
	i.app.SetFocus(i.configView)
}

func (i *Configurator) selectConfigs() {
	c, err := i.gConfigsList.GetCurrentConfigs(context.Background(), &empty.Empty{})
	if err != nil {
		i.configView.SetText(err.Error())
		return
	}
	i.configView.SetText(fmt.Sprintf("%+v", c))
	i.app.SetFocus(i.configView)
}

func (i *Configurator) selectThermal() {
	t, err := i.gThermal.GetCurrentProfile(context.Background(), &empty.Empty{})
	if err != nil {
		i.configView.SetText(err.Error())
		return
	}
	i.configView.SetText(fmt.Sprintf("%+v", t))
	i.app.SetFocus(i.configView)
}

func (i *Configurator) selectKeyboard() {
	k, err := i.gKeyboard.GetCurrentBrightness(context.Background(), &empty.Empty{})
	if err != nil {
		i.configView.SetText(err.Error())
		return
	}
	i.configView.SetText(fmt.Sprintf("%+v", k))
	i.app.SetFocus(i.configView)
}

func (i *Configurator) selectBattery() {
	b, err := i.gBattery.GetCurrentLimit(context.Background(), &empty.Empty{})
	if err != nil {
		i.configView.SetText(err.Error())
		return
	}
	i.configView.SetText(fmt.Sprintf("%+v", b))
	i.app.SetFocus(i.configView)
}

func (i *Configurator) Serve(haltCtx context.Context) error {

	i.setup()

	i.ctx, i.cancelFn = context.WithCancel(haltCtx)
	defer i.cancelFn()

	go func() {
		i.app.Run()
		i.cancelFn()
	}()

	for {
		select {
		case <-i.ctx.Done():
			i.app.Stop()
			return nil
		}
	}
}
