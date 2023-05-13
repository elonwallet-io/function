package server

import (
	"crypto/tls"
	"fmt"
	"github.com/rs/zerolog/log"
	"sync/atomic"
	"time"
)

type CertificateCache struct {
	certFilePath string
	keyFilePath  string
	certificate  atomic.Pointer[tls.Certificate]
	ticker       *time.Ticker
}

func (c *CertificateCache) loadCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(c.certFilePath, c.keyFilePath)
}

func (c *CertificateCache) refreshCertificate() {
	cert, err := c.loadCertificateFromFile()
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to refresh certificate")
	}

	c.certificate.Swap(&cert)
}

func (c *CertificateCache) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return c.certificate.Load(), nil
}

func NewCertificateCache(certFile, keyFile string) (*CertificateCache, error) {
	c := &CertificateCache{
		certFilePath: certFile,
		keyFilePath:  keyFile,
		certificate:  atomic.Pointer[tls.Certificate]{},
		ticker:       time.NewTicker(24 * time.Hour),
	}

	cert, err := c.loadCertificateFromFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate from file: %w", err)
	}
	c.certificate.Store(&cert)

	go func() {
		for {
			select {
			case <-c.ticker.C:
				c.refreshCertificate()
			}
		}
	}()

	return c, nil
}
