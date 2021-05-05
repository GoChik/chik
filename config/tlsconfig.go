package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"

	"github.com/smallstep/certificates/api"
	"github.com/smallstep/certificates/ca"
	"go.step.sm/crypto/pemutil"
)

const (
	certFile = "client.cert"
	keyFile  = "client.key"
)

func TlsConfig(ctx context.Context, token string) (*tls.Config, error) {
	if conf.currentPath == "" {
		return nil, errors.New("config error: config path is not set. Call config.ParseConfig first")
	}

	client, err := ca.Bootstrap(token)
	if err != nil {
		return nil, err
	}

	if !hasLocalKeyPair() {
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

	cert, err := tls.LoadX509KeyPair(filepath.Join(conf.currentPath, certFile), filepath.Join(conf.currentPath, keyFile))
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}

	renewer, err := ca.NewTLSRenewer(&cert, getRenewFunction(ctx, client, token))
	renewer.RunContext(ctx)

	return &tls.Config{
		RootCAs:              client.GetRootCAs(),
		InsecureSkipVerify:   false,
		GetClientCertificate: renewer.GetClientCertificate,
		GetCertificate:       renewer.GetCertificate,
	}, nil
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

func getRenewFunction(ctx context.Context, client *ca.Client, token string) func() (*tls.Certificate, error) {
	return func() (*tls.Certificate, error) {
		req, pk, err := ca.CreateSignRequest(token)
		if err != nil {
			return nil, err
		}

		resp, err := client.Sign(req)
		if err != nil {
			return nil, err
		}

		tr, err := client.Transport(ctx, resp, pk)
		if err != nil {
			return nil, err
		}
		// Renew the certificate. The return type is equivalent to the Sign method.
		renew, err := client.Renew(tr)
		saveCertificate(renew)

		cert, err := tls.LoadX509KeyPair(filepath.Join(conf.currentPath, certFile), filepath.Join(conf.currentPath, keyFile))
		return &cert, err
	}
}
