package chik

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gochik/chik/config"
	"github.com/rs/zerolog/log"
)

type RouterCloser = func() error

func StartRouter(ctx context.Context, controller *Controller) (RouterCloser, error) {
	var token string
	var port uint16

	for k, v := range map[string]interface{}{
		"connection.token": &token,
		"connection.port":  &port,
	} {
		err := config.GetStruct(k, v)
		if err != nil {
			log.Err(err).Msg("cannot start router")
			return nil, err
		}
	}

	conf, err := config.TLSConfig(ctx, token)
	if err != nil {
		log.Err(err).Msgf("Failed to get TLS config")
		return nil, err
	}

	server, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port), conf)
	if err != nil {
		log.Err(err).Msgf("Error starting server")
		return nil, err
	}

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				log.Err(err).Msg("connection failed")
				if errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}

			// Creating the controller that is handling the newly connected client
			go func() {
				log.Info().Str("client", conn.RemoteAddr().String()).Msg("New connection")
				tlsConn := conn.(*tls.Conn)
				err := tlsConn.Handshake()
				if err != nil {
					log.Err(err).Msg("Handshake failed.")
					tlsConn.Close()
					return
				}

				time.Sleep(5 * time.Millisecond) // TODO: we need to be sure handlers are started before being able to receive messages
				ctx, remoteCancel := StartRemote(controller, conn, MaxIdleTime)
				<-ctx.Done()
				remoteCancel()
			}()
		}
	}()
	return server.Close, nil
}
