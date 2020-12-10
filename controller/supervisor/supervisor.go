package supervisor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/server"

	"github.com/thejerf/suture"
)

type ManagerResponderOption struct {
	Supervisor       *suture.Supervisor
	ReloadCh         chan *controller.Dependencies
	Dependencies     *controller.Dependencies
	ManagerReqCh     chan server.ManagerSupervisorRequest
	ControllerConfig controller.RunConfig

	childToken        suture.ServiceToken
	controllerRunning bool
}

func ManagerResponder(haltCtx context.Context, opt ManagerResponderOption) {
	for {
		select {
		case s := <-opt.ManagerReqCh:
			switch s.Request {

			case server.RequestStartController:
				doStartController(s, &opt)

			case server.RequestStopController:
				doStopController(s, &opt)

			case server.RequestCheckState:
				if opt.controllerRunning {
					s.Response <- server.ManagerSupervisorResponse{
						Error: nil,
						State: protocol.ManagerControlResponse_RUNNING,
					}
				} else {
					s.Response <- server.ManagerSupervisorResponse{
						Error: nil,
						State: protocol.ManagerControlResponse_STOPPED,
					}
				}

			}
		case <-haltCtx.Done():
			log.Println("[supervisor] exiting grpcManagerResponder")
			return
		}
	}
}

func doStartController(s server.ManagerSupervisorRequest, opt *ManagerResponderOption) {
	if opt.controllerRunning {
		s.Response <- server.ManagerSupervisorResponse{
			Error: fmt.Errorf("Controller is already running"),
			State: protocol.ManagerControlResponse_RUNNING,
		}
		return
	}

	control, controllerStartErrCh, err := controller.New(opt.ControllerConfig, opt.Dependencies)
	if err != nil {
		s.Response <- server.ManagerSupervisorResponse{
			Error: err,
			State: protocol.ManagerControlResponse_STOPPED,
		}
		return
	}

	controllerSupervisor := suture.New("Controller", suture.Spec{})
	controllerSupervisor.Add(control)
	opt.childToken = opt.Supervisor.Add(controllerSupervisor)

	select {
	case controllerStartErr := <-controllerStartErrCh:
		s.Response <- server.ManagerSupervisorResponse{
			Error: controllerStartErr,
			State: protocol.ManagerControlResponse_STOPPED,
		}
		return
	case <-time.After(time.Second * 2):
		opt.controllerRunning = true
		s.Response <- server.ManagerSupervisorResponse{
			Error: nil,
			State: protocol.ManagerControlResponse_RUNNING,
		}
	}
}

func doStopController(s server.ManagerSupervisorRequest, opt *ManagerResponderOption) {

	if opt.controllerRunning {
		err := opt.Supervisor.Remove(opt.childToken)
		if err != nil {
			s.Response <- server.ManagerSupervisorResponse{
				Error: err,
				State: protocol.ManagerControlResponse_STOPPED,
			}
		} else {
			opt.controllerRunning = false
			s.Response <- server.ManagerSupervisorResponse{
				Error: nil,
				State: protocol.ManagerControlResponse_STOPPED,
			}
		}
	} else {
		s.Response <- server.ManagerSupervisorResponse{
			Error: fmt.Errorf("Controller is not running"),
			State: protocol.ManagerControlResponse_STOPPED,
		}
	}
}
