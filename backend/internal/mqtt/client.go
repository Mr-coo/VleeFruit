// Package mqtt connects the backend to the broker (Mosquitto). The backend
// authenticates as a privileged account allowed to see all device topics.
package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	paho "github.com/eclipse/paho.mqtt.golang"
)

// newClientOptions builds paho options from config, including TLS when the
// broker URL is a tls:// endpoint. onConnect runs on every (re)connection so
// subscriptions survive reconnects.
func newClientOptions(cfg config.MQTTConfig, onConnect paho.OnConnectHandler) (*paho.ClientOptions, error) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetOnConnectHandler(onConnect)

	tlsCfg, err := tlsConfig(cfg)
	if err != nil {
		return nil, err
	}
	if tlsCfg != nil {
		opts.SetTLSConfig(tlsCfg)
	}
	return opts, nil
}

// tlsConfig returns a *tls.Config when a CA file is provided or insecure mode
// is requested; otherwise nil (plain TCP).
func tlsConfig(cfg config.MQTTConfig) (*tls.Config, error) {
	if cfg.CAFile == "" && !cfg.Insecure {
		return nil, nil
	}
	t := &tls.Config{InsecureSkipVerify: cfg.Insecure} //nolint:gosec // dev opt-in
	if cfg.CAFile != "" {
		pem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read mqtt CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("parse mqtt CA: no certs found")
		}
		t.RootCAs = pool
	}
	return t, nil
}
