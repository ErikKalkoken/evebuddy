// Package remoteservice provides a service for communicating between instances remotely.
package remoteservice

import (
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
)

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
func (sw RemoteService) ShowInstance(request string, reply *string) error {
	sw.showInstance()
	slog.Info("Remote Service: ShowInstance completed")
	return nil
}

// Start starts the remote service.
func Start(port int, showInstance func()) error {
	err := rpc.Register(newRemoteService(showInstance))
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("remote service: listen error: %w", err)
	}
	go func() {
		defer listener.Close()
		slog.Info("Remote service running", "port", port)
		for {
			conn, err := listener.Accept()
			if err != nil {
				slog.Error("remote service: Failed to accept connection", "err", err)
				return
			}
			go rpc.ServeConn(conn)
		}
	}()
	return nil
}

// ShowPrimaryInstance sends a request to the primary instance to show it.
// This function should be called by a secondary instance.
func ShowPrimaryInstance(port int) error {
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return fmt.Errorf("dial remote service: %w", err)
	}
	defer client.Close()
	var reply string
	err = client.Call("RemoteService.ShowInstance", "", &reply)
	if err != nil {
		return fmt.Errorf("call remote service: %w", err)
	}
	slog.Info("RemoteService.ShowInstance called")
	return nil
}
