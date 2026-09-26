// Package mqtt provides an MQTT layer that runs alongside the Gin HTTP server.
// It connects to a broker, replies "ok" on a health request topic, and analyzes
// images published to the image topic via the vision LLM.
package mqtt

import (
	"context"
	"log"
	"net/http"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// ImageAnalyzer analyzes raw image bytes and returns a text result. It is
// satisfied by *llm.Client.
type ImageAnalyzer interface {
	Analyze(ctx context.Context, imageData []byte, mimeType string) (string, error)
}

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
	// RequestTopic is the health topic this service subscribes to (replies "ok").
	RequestTopic string
	// ResponseTopic is where health replies are published.
	ResponseTopic string
	// ImageRequestTopic is where devices publish image bytes to be analyzed.
	ImageRequestTopic string
	// ImageResponseTopic is where the LLM analysis is published.
	ImageResponseTopic string
	// QoS is the quality-of-service level for subscribe and publish (0, 1 or 2).
	QoS byte
}

// Client is a thin wrapper around the Paho MQTT client.
type Client struct {
	cfg      Config
	analyzer ImageAnalyzer
	client   paho.Client
}

// NewClient builds an MQTT client, applying sensible defaults. The analyzer is
// used to handle images; it may be nil to disable image handling. It does not
// connect; call Start for that.
func NewClient(cfg Config, analyzer ImageAnalyzer) *Client {
	if cfg.ClientID == "" {
		cfg.ClientID = "vleefruit-backend"
	}
	if cfg.RequestTopic == "" {
		cfg.RequestTopic = "vleefruit/request"
	}
	if cfg.ResponseTopic == "" {
		cfg.ResponseTopic = "vleefruit/response"
	}
	if cfg.ImageRequestTopic == "" {
		cfg.ImageRequestTopic = "vleefruit/image/request"
	}
	if cfg.ImageResponseTopic == "" {
		cfg.ImageResponseTopic = "vleefruit/image/response"
	}
	return &Client{cfg: cfg, analyzer: analyzer}
}

// Enabled reports whether a broker is configured.
func (c *Client) Enabled() bool { return c.cfg.BrokerURL != "" }

// Start connects to the broker and subscribes to the topics. When MQTT is not
// configured it logs and returns nil so the server keeps running.
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
	subs := map[string]paho.MessageHandler{
		c.cfg.RequestTopic:      c.handleHealth,
		c.cfg.ImageRequestTopic: c.handleImage,
	}
	for topic, handler := range subs {
		if token := client.Subscribe(topic, c.cfg.QoS, handler); token.Wait() && token.Error() != nil {
			log.Printf("mqtt: subscribe to %q failed: %v", topic, token.Error())
			continue
		}
		log.Printf("mqtt: subscribed to %q", topic)
	}
}

// handleHealth is the minimal handler: reply "ok" on the response topic.
func (c *Client) handleHealth(client paho.Client, msg paho.Message) {
	log.Printf("mqtt: message on %q (%d bytes)", msg.Topic(), len(msg.Payload()))
	client.Publish(c.cfg.ResponseTopic, c.cfg.QoS, false, "ok")
}

// handleImage runs the payload (raw image bytes) through the vision LLM and
// publishes the result on the image response topic.
func (c *Client) handleImage(client paho.Client, msg paho.Message) {
	data := msg.Payload()
	log.Printf("mqtt: image on %q (%d bytes)", msg.Topic(), len(data))

	if c.analyzer == nil {
		c.publishImageResult(client, "error: image analysis not configured")
		return
	}
	if len(data) == 0 {
		c.publishImageResult(client, "error: empty image payload")
		return
	}

	// Detect the content type from the payload's magic bytes.
	mimeType := http.DetectContentType(data)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := c.analyzer.Analyze(ctx, data, mimeType)
	if err != nil {
		log.Printf("mqtt: image analysis failed: %v", err)
		c.publishImageResult(client, "error: "+err.Error())
		return
	}
	c.publishImageResult(client, result)
}

func (c *Client) publishImageResult(client paho.Client, payload string) {
	client.Publish(c.cfg.ImageResponseTopic, c.cfg.QoS, false, payload)
}

// Stop disconnects from the broker. Safe to call when never started.
func (c *Client) Stop() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
		log.Printf("mqtt: disconnected")
	}
}
