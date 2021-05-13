package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/smallstep/certificates/api"
	"github.com/smallstep/certificates/ca"
	"go.step.sm/crypto/jose"
	"go.step.sm/crypto/pemutil"
)

const (
	certFile = "client.cert"
	keyFile  = "client.key"
)

var logger = log.With().Str("module", "tls_config").Logger()

type tokenClaims struct {
	RootSHA string `json:"sha"`
	jose.Claims
}

func TLSConfig(ctx context.Context, token string) (*tls.Config, error) {
	if conf.currentPath == "" {
		return nil, errors.New("config error: config path is not set. Call config.ParseConfig first")
	}

	claims, err := parseToken(token)
	if err != nil {
		return nil, fmt.Errorf("parse token failed: %v", err)
	}

	if !hasLocalKeyPair() {
		client, err := ca.NewClient(claims.Audience[0], ca.WithRootSHA256(claims.RootSHA))
		if err != nil {
			return nil, err
		}

		req, pk, err := ca.CreateSignRequest(token)
		if err != nil {
			return nil, err
		}

		pemutil.Serialize(pk, pemutil.ToFile(filepath.Join(conf.currentPath, keyFile), 0600))

		// Get the certificate
		resp, err := client.Sign(req)
		if err != nil {
			return nil, err
		}

		err = saveCertificate(resp)
		if err != nil {
			return nil, err
		}
	}

	cert, _ := loadCertificate()

	client, err := ca.NewClient(claims.Audience[0], ca.WithCertificate(cert), ca.WithRootSHA256(claims.RootSHA))
	if err != nil {
		return nil, err
	}

	renewer, err := ca.NewTLSRenewer(&cert, getRenewFunction(ctx, claims))
	renewer.RunContext(ctx)

	return &tls.Config{
		RootCAs:              client.GetRootCAs(),
		InsecureSkipVerify:   false,
		GetClientCertificate: renewer.GetClientCertificate,
		GetCertificate:       renewer.GetCertificate,
	}, nil
}

func parseToken(token string) (claims tokenClaims, err error) {
	tok, err := jose.ParseSigned(token)
	if err != nil {
		return
	}

	if err = tok.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return
	}

	// Validate bootstrap token
	switch {
	case len(claims.RootSHA) == 0:
		err = errors.New("invalid bootstrap token: sha claim is not present")
		return
	case !strings.HasPrefix(strings.ToLower(claims.Audience[0]), "http"):
		err = errors.New("invalid bootstrap token: aud claim is not a url")
		return
	}
	return
}

func hasLocalKeyPair() bool {
	for _, file := range []string{certFile, keyFile} {
		_, err := os.Stat(filepath.Join(conf.currentPath, file))
		if err != nil {
			return false
		}
	}

	return true
}

func saveCertificate(resp *api.SignResponse) error {
	certPEM, err := pemutil.Serialize(resp.ServerPEM.Certificate)
	if err != nil {
		return err
	}
	caPEM, err := pemutil.Serialize(resp.CaPEM.Certificate)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(conf.currentPath, certFile))
	if err != nil {
		return err
	}
	defer f.Close()

	err = pem.Encode(f, certPEM)
	if err != nil {
		return err
	}

	err = pem.Encode(f, caPEM)
	return err
}

func loadCertificate() (cert tls.Certificate, err error) {
	cert, err = tls.LoadX509KeyPair(filepath.Join(conf.currentPath, certFile), filepath.Join(conf.currentPath, keyFile))
	if err != nil {
		return
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	return
}

func getRenewFunction(ctx context.Context, claims tokenClaims) func() (*tls.Certificate, error) {
	return func() (*tls.Certificate, error) {
		cert, _ := loadCertificate()

		client, err := ca.NewClient(claims.Audience[0], ca.WithCertificate(cert), ca.WithRootSHA256(claims.RootSHA))
		if err != nil {
			return nil, err
		}

		dialer := net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		transport := http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				RootCAs:            client.GetRootCAs(),
				InsecureSkipVerify: false,
				Certificates:       []tls.Certificate{cert},
			},
		}

		// Renew the certificate. The return type is equivalent to the Sign method.
		renew, err := client.Renew(&transport)
		if err != nil {
			logger.Err(err).Msg("Cannot renew certificate")
			return nil, err
		}
		saveCertificate(renew)

		cert, err = loadCertificate()
		return &cert, err
	}
}
