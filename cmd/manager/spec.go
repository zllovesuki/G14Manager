package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zllovesuki/G14Manager/controller"
	"github.com/zllovesuki/G14Manager/rpc/protocol"
	"github.com/zllovesuki/G14Manager/rpc/server"
	"github.com/zllovesuki/G14Manager/util"

	"cirello.io/oversight"
)

const (
	controllerChildName = "Controller"
)

func getControllerSpec(s server.SupervisorRequest, config controller.RunConfig, dep *controller.Dependencies) oversight.ChildProcessSpecification {
	return oversight.ChildProcessSpecification{
		Name: controllerChildName,
		Start: func(ctx context.Context) error {
			control, err := controller.New(config, dep)
			if err != nil {
				s.Response <- server.SupervisorResponse{
					Error: err,
					State: protocol.ManagerControlResponse_STOPPED,
				}
				return nil
			}

			errCh := make(chan error)
			go func() {
				errCh <- control.Run(ctx)
			}()

			select {
			case err := <-errCh:
				s.Response <- server.SupervisorResponse{
					Error: err,
					State: protocol.ManagerControlResponse_STOPPED,
				}
				return err
			case <-time.After(time.Second * 2):
				s.Response <- server.SupervisorResponse{
					Error: nil,
					State: protocol.ManagerControlResponse_RUNNING,
				}
				return <-errCh
			}
		},
		Restart: func(err error) bool {
			if err != nil {
				log.Println("[supervisor] controller returned an error:")
				log.Printf("%+v\n", err)
				util.SendToastNotification("G14Manager Supervisor", util.Notification{
					Title:   "G14Manager needs to be restarted",
					Message: fmt.Sprintf("An error has occurred: %s", err),
				})
			}
			return false
		},
	}
}

func getGRPCSpec(s *controller.Server) oversight.ChildProcessSpecification {
	return oversight.ChildProcessSpecification{
		Name: "gRPCServer",
		Start: func(ctx context.Context) error {
			return s.Run(ctx)
		},
	}
}
