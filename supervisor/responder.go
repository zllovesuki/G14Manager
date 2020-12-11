package supervisor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/rpc/server"

	suture "github.com/thejerf/suture/v4"
)

type controllerState int

const (
	controllerStopped controllerState = iota
	controllerRunning
	controllerUnknown
)

type ManagerResponderOption struct {
	Supervisor       *suture.Supervisor
	Dependencies     *controller.Dependencies
	ManagerReqCh     chan server.ManagerSupervisorRequest
	ControllerConfig controller.RunConfig

	childToken      suture.ServiceToken
	controllerState controllerState
}

func (m *ManagerResponderOption) HasSupervisor() *suture.Supervisor {
	return m.Supervisor
}

func (m *ManagerResponderOption) String() string {
	return "ManagerResponder"
}

func (m *ManagerResponderOption) Serve(haltCtx context.Context) error {

	log.Println("[gRPCSupervisor] starting responder loop")

	for {
		select {
		case s := <-m.ManagerReqCh:
			switch s.Request {

			case server.RequestStartController:
				m.doStartController(s)

			case server.RequestStopController:
				m.doStopController(s)

			case server.RequestCheckState:
				switch m.controllerState {
				case controllerRunning:
					s.Response <- server.ManagerSupervisorResponse{
						Error: nil,
						State: protocol.ManagerControlResponse_RUNNING,
					}
				case controllerStopped:
					s.Response <- server.ManagerSupervisorResponse{
						Error: nil,
						State: protocol.ManagerControlResponse_STOPPED,
					}
				case controllerUnknown:
					s.Response <- server.ManagerSupervisorResponse{
						Error: nil,
						State: protocol.ManagerControlResponse_UNKNOWN,
					}
				}

			case server.RequestSaveConfig:
				err := m.Dependencies.ConfigRegistry.Save()
				s.Response <- server.ManagerSupervisorResponse{
					Error: err,
				}

			}
		case <-haltCtx.Done():
			log.Println("[gRPCSupervisor] exiting ManagerResponder")
			return nil
		}
	}
}

func (m *ManagerResponderOption) doStartController(s server.ManagerSupervisorRequest) {
	switch m.controllerState {
	case controllerRunning:
		s.Response <- server.ManagerSupervisorResponse{
			Error: fmt.Errorf("Controller is already running"),
			State: protocol.ManagerControlResponse_RUNNING,
		}
		return
	case controllerUnknown:
		s.Response <- server.ManagerSupervisorResponse{
			Error: fmt.Errorf("Controller is in unknown state"),
			State: protocol.ManagerControlResponse_UNKNOWN,
		}
		return
	}

	control, controllerStartErrCh, err := controller.New(m.ControllerConfig, m.Dependencies)
	if err != nil {
		s.Response <- server.ManagerSupervisorResponse{
			Error: err,
			State: protocol.ManagerControlResponse_STOPPED,
		}
		return
	}

	controllerSupervisor := suture.New("controllerSupervisor", suture.Spec{})
	controllerSupervisor.Add(control)
	m.childToken = m.Supervisor.Add(controllerSupervisor)

	select {
	case controllerStartErr := <-controllerStartErrCh:
		s.Response <- server.ManagerSupervisorResponse{
			Error: controllerStartErr,
			State: protocol.ManagerControlResponse_STOPPED,
		}
		return
	case <-time.After(time.Second * 2):
		m.controllerState = controllerRunning
		s.Response <- server.ManagerSupervisorResponse{
			Error: nil,
			State: protocol.ManagerControlResponse_RUNNING,
		}
	}
}

func (m *ManagerResponderOption) doStopController(s server.ManagerSupervisorRequest) {
	switch m.controllerState {
	case controllerStopped:
		s.Response <- server.ManagerSupervisorResponse{
			Error: fmt.Errorf("Controller is not running"),
			State: protocol.ManagerControlResponse_RUNNING,
		}
		return
	case controllerUnknown:
		s.Response <- server.ManagerSupervisorResponse{
			Error: fmt.Errorf("Controller is in unknown state"),
			State: protocol.ManagerControlResponse_UNKNOWN,
		}
		return
	}

	err := m.Supervisor.RemoveAndWait(m.childToken, time.Second*2)
	if err != nil {
		s.Response <- server.ManagerSupervisorResponse{
			Error: err,
			State: protocol.ManagerControlResponse_UNKNOWN,
		}
	} else {
		m.controllerState = controllerStopped
		s.Response <- server.ManagerSupervisorResponse{
			Error: nil,
			State: protocol.ManagerControlResponse_STOPPED,
		}
	}
}
