package client

import (
	"context"
	"fmt"
	"strconv"
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

	frame                *tview.Frame
	container            *tview.Flex
	containerLeftCol     *tview.Flex
	containerRightCol    *tview.Flex
	containerPlaceholder *tview.Box

	configEditHolder *tview.Flex
	configEdit       *tview.TextView
	configView       *tview.TextView
	infoView         *tview.TextView

	batteryEdit *tview.Form

	fnLists     *tview.List
	fnListItems []listItem

	dataBinding data
}

type data struct {
	battery uint32
}

type listItem struct {
	Main            string
	Secondary       string
	Shortcut        rune
	Callback        func()
	EditPrimitive   tview.Primitive
	HideEditTooltip bool
}

func NewInterface() *Configurator {
	return &Configurator{
		app:                  tview.NewApplication(),
		layers:               tview.NewPages(),
		connectModal:         tview.NewModal(),
		confirmationModal:    tview.NewModal(),
		container:            tview.NewFlex(),
		containerLeftCol:     tview.NewFlex(),
		containerRightCol:    tview.NewFlex(),
		containerPlaceholder: tview.NewBox(),
		configEditHolder:     tview.NewFlex(),
		configEdit:           tview.NewTextView(),
		configView:           tview.NewTextView(),
		infoView:             tview.NewTextView(),
		batteryEdit:          tview.NewForm(),
		fnLists:              tview.NewList(),
		dataBinding:          data{},
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

	i.updateInfoView()
	return nil
}

func (i *Configurator) setup() {
	i.layers.
		AddPage("connect", i.connectModal, true, true).
		AddPage("container", i.container, true, false).
		AddPage("confirmation", i.confirmationModal, true, false)

	i.setupFnList()
	i.setupModals()

	i.setupForms()

	i.setupStyles()
	i.keyBindings()

	i.containerLeftCol.SetDirection(tview.FlexRow).
		AddItem(i.fnLists, 0, 8, true).
		AddItem(i.infoView, 0, 2, false)

	i.containerRightCol.SetDirection(tview.FlexRow).
		AddItem(i.configView, 0, 2, false).
		AddItem(i.configEditHolder, 0, 4, false)

	i.container.
		AddItem(i.containerLeftCol, 0, 3, true).
		AddItem(i.containerRightCol, 0, 7, false)

	i.configView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyESC {
			i.app.SetFocus(i.fnLists)
		}
	}).Focus(func(p tview.Primitive) {
		i.showMessage(fmt.Sprintf("%T", p), tcell.ColorRed)
	})

	i.frame = tview.NewFrame(i.layers)

	i.clearMessage()

	i.app.SetRoot(i.frame, true)
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
			Secondary: "Get/Set features & profiles",
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
			Main:      "Keyboard Backlight",
			Secondary: "Get/Set backlight level",
			Shortcut:  'k',
			Callback:  i.selectKeyboard,
		},
		{
			Main:          "Battery Charge Limit",
			Secondary:     "Get/Set charge limit",
			Shortcut:      'b',
			Callback:      i.selectBattery,
			EditPrimitive: i.batteryEdit,
		},
		{
			Main:      "Exit",
			Secondary: "Exit the Configurator",
			Shortcut:  'q',
			Callback: func() {
				i.confirmNo = "container"
				i.confirmYes = i.cancelFn
				i.layers.SwitchToPage("confirmation")
			},
		},
	}
	for index := range i.fnListItems {
		item := i.fnListItems[index]
		i.fnLists.AddItem(item.Main, item.Secondary, item.Shortcut, item.Callback)
	}
}

func (i *Configurator) clearConfigEdit() {
	i.configEditHolder.Clear()
	i.app.SetFocus(i.configView)
}

func (i *Configurator) setupForms() {
	i.batteryEdit.
		AddInputField("New Charge Limit Percentage ", "", 20, func(textToCheck string, lastChar rune) bool {
			_, err := strconv.ParseUint(textToCheck, 10, 32)
			if err != nil {
				return false
			}
			// i.showError(textToCheck)
			return true
		}, nil).
		AddButton("Cancel", func() {
			i.clearConfigEdit()
			i.showEditTooltip()
		}).
		AddButton("Save", func() {
			num, _ := strconv.ParseUint(i.batteryEdit.GetFormItem(0).(*tview.InputField).GetText(), 10, 32)
			i.dataBinding.battery = uint32(num)
			b, err := i.gBattery.Set(context.Background(), &protocol.SetBatteryLimitRequest{
				Percentage: uint32(num),
			})
			if err != nil {
				i.showMessage(err.Error(), tcell.ColorRed)
				return
			}

			if b.GetSuccess() == false {
				i.showMessage(b.GetMessage(), tcell.ColorRed)
				return
			}

			i.showMessage("Charge limit updated!", tcell.ColorGreen)
			i.clearConfigEdit()
			i.selectBattery()
		}).
		SetButtonBackgroundColor(tcell.Color104).
		SetFieldBackgroundColor(tcell.Color104)
}

func (i *Configurator) keyBindings() {
	// Right key on function list will select the item
	i.fnLists.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRight {
			item := i.fnListItems[i.fnLists.GetCurrentItem()]
			item.Callback()
			return nil
		}
		return event
	})

	// Left key on configView will go back to function list
	i.configView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyEsc {
			i.clearMessage()
			i.app.SetFocus(i.fnLists)
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'e' {
			i.configEditHolder.Clear()
			currentItem := i.fnLists.GetCurrentItem()
			editPrim := i.fnListItems[currentItem].EditPrimitive

			if editPrim != nil {
				i.clearMessage()
				i.configEditHolder.AddItem(editPrim, 0, 1, true)
				i.app.SetFocus(editPrim)
			}
		}
		return event
	})
}

func (i *Configurator) setupStyles() {
	i.fnLists.Box.SetBorder(true).SetTitle(" Functions ")
	i.fnLists.SetSecondaryTextColor(tcell.ColorGray)
	i.configView.Box.SetBorder(true).SetBorderAttributes(tcell.AttrNone).SetTitle(" Current Settings ")
	i.configEdit.Box.SetBorder(true).SetTitle(" Edit Settings ")
	i.infoView.Box.SetBorder(true).SetTitle(" Information ")
}

func (i *Configurator) updateInfoView() {
	var txt string
	txt = fmt.Sprintf("%v\nSupervisor version: %s", txt, "blah")
	txt = fmt.Sprintf("%v\nClient version: %s", txt, "blahblah")
	txt = fmt.Sprintf("%v\nLogs: 127.0.0.1:9969/debug/logs", txt)
	i.infoView.SetText(txt[1:])
}

func (i *Configurator) showEditTooltip() {
	i.frame.Clear().AddText("G14Manager Configurator", true, tview.AlignCenter, tcell.ColorWhite).AddText("Press (E) to edit", false, tview.AlignLeft, tcell.ColorWhite)
}

func (i *Configurator) clearMessage() {
	i.frame.Clear().AddText("G14Manager Configurator", true, tview.AlignCenter, tcell.ColorWhite).AddText("", false, tview.AlignLeft, tcell.ColorWhite)
}

func (i *Configurator) showMessage(msg string, color tcell.Color) {
	i.frame.Clear().AddText("G14Manager Configurator", true, tview.AlignCenter, tcell.ColorWhite).AddText(msg, false, tview.AlignLeft, color)
	go func() {
		time.Sleep(time.Millisecond * 2500)
		i.clearMessage()
		i.app.Draw()
	}()
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
		return
	}
	var txt string
	if auto.Success != true {
		txt = fmt.Sprintf("%sCannot get AutoStart status: %s\n\n", txt, auto.Message)
	} else {
		txt = fmt.Sprintf("%sRun Controller when Supervisor starts: %t\n\n", txt, auto.AutoStart)
	}
	if m.Success != true {
		txt = fmt.Sprintf("%sCannot get Manager running state: %s\n\n", txt, m.Message)
	} else {
		txt = fmt.Sprintf("%sController is currently: %s\n\n", txt, m.State)
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
		i.showMessage(err.Error(), tcell.ColorRed)
		return
	}

	if b.GetSuccess() == false {
		i.showMessage(b.GetMessage(), tcell.ColorRed)
		return
	}

	i.dataBinding.battery = b.GetPercentage()

	var txt string
	txt = fmt.Sprintf("%sCurrent battery charge limit: %d%%\n", txt, b.GetPercentage())
	i.configView.SetText(txt)

	i.batteryEdit.GetFormItem(0).(*tview.InputField).SetText(fmt.Sprintf("%d", i.dataBinding.battery))

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
