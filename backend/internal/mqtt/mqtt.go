// Package mqtt provides a minimal MQTT layer that runs alongside the Gin HTTP
// server. For now it connects to a broker, subscribes to a request topic, and
// replies "ok" on the matching response topic.
package mqtt

import (
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Config configures the MQTT client. An empty BrokerURL disables the layer so
// the server still starts.
type Config struct {
	// BrokerURL is the broker address, e.g. "tcp://localhost:1883". Empty disables MQTT.
	BrokerURL string
	// ClientID identifies this connection to the broker.
	ClientID string
	// Username / Password are optional broker credentials.
	Username string
	Password string
	// RequestTopic is the topic this service subscribes to.
	RequestTopic string
	// ResponseTopic is where replies are published.
	ResponseTopic string
	// QoS is the quality-of-service level for subscribe and publish (0, 1 or 2).
	QoS byte
}

// Client is a thin wrapper around the Paho MQTT client.
type Client struct {
	cfg    Config
	client paho.Client
}

// NewClient builds an MQTT client, applying sensible defaults. It does not
// connect; call Start for that.
func NewClient(cfg Config) *Client {
	if cfg.ClientID == "" {
		cfg.ClientID = "vleefruit-backend"
	}
	if cfg.RequestTopic == "" {
		cfg.RequestTopic = "vleefruit/request"
	}
	if cfg.ResponseTopic == "" {
		cfg.ResponseTopic = "vleefruit/response"
	}
	return &Client{cfg: cfg}
}

// Enabled reports whether a broker is configured.
func (c *Client) Enabled() bool { return c.cfg.BrokerURL != "" }

// Start connects to the broker and subscribes to the request topic. When MQTT
// is not configured it logs and returns nil so the server keeps running.
func (c *Client) Start() error {
	if !c.Enabled() {
		log.Printf("mqtt: no MQTT_BROKER_URL set; MQTT layer disabled")
		return nil
	}

	opts := paho.NewClientOptions().
		AddBroker(c.cfg.BrokerURL).
		SetClientID(c.cfg.ClientID).
		SetUsername(c.cfg.Username).
		SetPassword(c.cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second).
		SetOnConnectHandler(c.onConnect)

	c.client = paho.NewClient(opts)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("mqtt: connected to %s", c.cfg.BrokerURL)
	return nil
}

// onConnect (re)subscribes on every successful connection, including reconnects.
func (c *Client) onConnect(client paho.Client) {
	if token := client.Subscribe(c.cfg.RequestTopic, c.cfg.QoS, c.handleMessage); token.Wait() && token.Error() != nil {
		log.Printf("mqtt: subscribe to %q failed: %v", c.cfg.RequestTopic, token.Error())
		return
	}
	log.Printf("mqtt: subscribed to %q", c.cfg.RequestTopic)
}

// handleMessage is the minimal handler: reply "ok" on the response topic.
func (c *Client) handleMessage(client paho.Client, msg paho.Message) {
	log.Printf("mqtt: message on %q (%d bytes)", msg.Topic(), len(msg.Payload()))
	client.Publish(c.cfg.ResponseTopic, c.cfg.QoS, false, "ok")
}

// Stop disconnects from the broker. Safe to call when never started.
func (c *Client) Stop() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Printf("mqtt: disconnected")
	}
}
