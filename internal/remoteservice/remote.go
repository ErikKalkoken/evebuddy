// Package remoteservice provides a service for communicating between instances remotely.
package remoteservice

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
	"time"
)

const (
	callTimeout = 3 * time.Second
	dialTimeout = 3 * time.Second
)

type RemoteService struct {
	showInstance func()
}

func newRemoteService(showInstance func()) *RemoteService {
	if showInstance == nil {
		panic("showInstance can not be nil")
	}
	return &RemoteService{
		showInstance: showInstance,
	}
}

// ShowInstance shows the instance that is running the service.
func (sw *RemoteService) ShowInstance(args struct{}, reply *struct{}) error {
	sw.showInstance()
	slog.Info("Remote Service: ShowInstance completed")
	return nil
}

// Start starts the remote service.
func Start(port int, showInstance func()) (stop func(), err error) {
	server := rpc.NewServer()
	svc := newRemoteService(showInstance)
	if err := server.Register(svc); err != nil {
		return nil, fmt.Errorf("remote service: register error: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("remote service: listen error: %w", err)
	}

	go func() {
		slog.Info("Remote service running", "port", port)
		for {
			conn, err := listener.Accept()
			if errors.Is(err, net.ErrClosed) {
				slog.Info("remote service: closed")
				return
			}
			if err != nil {
				slog.Error("remote service: Failed to accept connection", "err", err)
				time.Sleep(50 * time.Millisecond) // transient error delay
				continue
			}
			go server.ServeConn(conn)
		}
	}()

	stop = func() {
		_ = listener.Close()
	}
	return stop, nil
}

// ShowPrimaryInstance sends a request to the primary instance to show it.
// This function should be called by a secondary instance.
func ShowPrimaryInstance(port int) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), dialTimeout)
	if err != nil {
		return fmt.Errorf("remote service dial error: %w", err)
	}
	defer conn.Close()

	// Enforce hard socket-level deadline for all RPC read/write ops
	if err := conn.SetDeadline(time.Now().Add(callTimeout)); err != nil {
		return fmt.Errorf("failed to set connection deadline: %w", err)
	}

	client := rpc.NewClient(conn)
	defer client.Close()

	var reply struct{}
	if err := client.Call("RemoteService.ShowInstance", struct{}{}, &reply); err != nil {
		return fmt.Errorf("call remote service error: %w", err)
	}

	slog.Info("RemoteService.ShowInstance called")
	return nil
}
