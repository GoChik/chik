package test

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/heartbeat"
	"github.com/gochik/chik/handlers/router"
	"github.com/gofrs/uuid"
)

type TestClient struct {
	remote *chik.Controller
	id     uuid.UUID
}

var address net.Addr
var peers = sync.Map{}
var serverID = uuid.Nil

func createController() *chik.Controller {
	f, err := ioutil.TempFile("", "tst")
	if err != nil {
		return nil
	}
	defer os.Remove(f.Name())

	config.AddSearchPath(os.TempDir())
	config.SetConfigFileName(filepath.Base(f.Name()))
	return chik.NewController()
}

func CreateServer(t *testing.T) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	address = listener.Addr()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			srv := createController()
			if srv == nil {
				t.Fatal("Cannot create controller")
			}
			serverID = srv.ID
			go func() {
				ctx, cancel := context.WithCancel(context.Background())
				go srv.Start(ctx, []chik.Handler{
					router.New(&peers),
					heartbeat.New(),
				})
				remote, _ := chik.StartRemote(srv, conn, 10*time.Second)
				<-remote.Done()
				cancel()
			}()
		}
	}()
	return listener
}

func CreateClient() (client TestClient, err error) {
	conn, err := net.Dial("tcp", address.String())
	if err != nil {
		return
	}
	controller := createController()
	if controller == nil {
		err = errors.New("failed to create a controller")
		return
	}
	chik.StartRemote(controller, conn, 10*time.Second)
	client = TestClient{controller, controller.ID}
	return
}
