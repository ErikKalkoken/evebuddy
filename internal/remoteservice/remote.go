// Package remoteservice provides a service for communicating between instances remotely.
package remoteservice

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
)

// TODO: Add tests

const (
	remoteServicePort = 30124
)

type EmptyArgs struct{}

// RemoteService is a RPC service allows to communicate between multiple instances of EVE Buddy.
type RemoteService struct {
	showInstance func() // callback that show an instance of the app
}

func newRemoteService(showInstance func()) *RemoteService {
	if showInstance == nil {
		panic("showInstance can not be nil")
	}
	s := &RemoteService{
		showInstance: showInstance,
	}
	return s
}

// ShowInstance shows the instance that is running the service.
func (sw RemoteService) ShowInstance(args *EmptyArgs, reply *EmptyArgs) error {
	sw.showInstance()
	slog.Info("Remote Service: ShowInstance completed")
	return nil
}

// Start starts the remote service.
func Start(showInstance func()) error {
	rpc.Register(newRemoteService(showInstance))
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", remoteServicePort))
	if err != nil {
		return fmt.Errorf("remote service: listen error: %w", err)
	}
	go func() {
		slog.Info("Remote service running", "port", remoteServicePort)
		err := http.Serve(l, nil)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Remote service: terminated prematurely", "error", err)
		}
	}()
	return nil
}

// ShowMainInstance sends a request the main instance to show it.
// This function should be called by the client.
func ShowMainInstance() error {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("localhost:%d", remoteServicePort))
	if err != nil {
		return fmt.Errorf("dial remote service: %w", err)
	}
	err = client.Call("RemoteService.ShowInstance", &EmptyArgs{}, &EmptyArgs{})
	if err != nil {
		return fmt.Errorf("call remote service: %w", err)
	}
	slog.Info("RemoteService.ShowInstance called")
	return nil
}
