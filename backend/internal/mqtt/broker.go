package mqtt

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
	"github.com/Mr-coo/VleeFruit/backend/internal/service"
	paho "github.com/eclipse/paho.mqtt.golang"
)

const (
	qosAtLeastOnce = 1
	handlerTimeout = 30 * time.Second
)

// Broker subscribes to device topics and drives the ingestion pipeline,
// publishing results back to each device.
type Broker struct {
	client        paho.Client
	ingestion     *service.IngestionService
	readings      *service.ReadingService
	devices       *repository.DeviceRepository
	maxImageBytes int64
}

func NewBroker(
	ingestion *service.IngestionService,
	readings *service.ReadingService,
	devices *repository.DeviceRepository,
) *Broker {
	return &Broker{ingestion: ingestion, readings: readings, devices: devices}
}

// Connect builds the client and starts connecting in the background. It does
// not fail if the broker is briefly unavailable — paho retries, and
// subscriptions are (re)established via the on-connect handler.
func (b *Broker) Connect(cfg config.MQTTConfig) error {
	b.maxImageBytes = cfg.MaxImageBytes
	opts, err := newClientOptions(cfg, b.onConnect)
	if err != nil {
		return err
	}
	b.client = paho.NewClient(opts)
	token := b.client.Connect()
	if token.WaitTimeout(15*time.Second) && token.Error() != nil {
		log.Printf("mqtt: initial connect error (will keep retrying): %v", token.Error())
	}
	return nil
}

// onConnect (re)subscribes on every successful connection.
func (b *Broker) onConnect(c paho.Client) {
	if token := c.Subscribe(topicImagesWildcard, qosAtLeastOnce, b.handleImage); token.Wait() && token.Error() != nil {
		log.Printf("mqtt: subscribe %s: %v", topicImagesWildcard, token.Error())
	}
	if token := c.Subscribe(topicReadingsWildcard, qosAtLeastOnce, b.handleReading); token.Wait() && token.Error() != nil {
		log.Printf("mqtt: subscribe %s: %v", topicReadingsWildcard, token.Error())
	}
	log.Printf("mqtt: connected and subscribed to %s, %s", topicImagesWildcard, topicReadingsWildcard)
}

// Stop disconnects the client.
func (b *Broker) Stop() {
	if b.client != nil {
		b.client.Disconnect(250)
	}
}

// handleImage ingests a raw image and publishes the detection result back.
func (b *Broker) handleImage(_ paho.Client, msg paho.Message) {
	deviceID, ok := deviceIDFromTopic(msg.Topic())
	if !ok {
		log.Printf("mqtt: bad image topic %q", msg.Topic())
		return
	}

	if b.maxImageBytes > 0 && int64(len(msg.Payload())) > b.maxImageBytes {
		log.Printf("mqtt: image from %s rejected: %d bytes exceeds limit %d",
			deviceID, len(msg.Payload()), b.maxImageBytes)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeout)
	defer cancel()

	b.touch(ctx, deviceID)

	result, err := b.ingestion.IngestImage(ctx, deviceID, msg.Payload())
	if err != nil {
		log.Printf("mqtt: ingest image from %s: %v", deviceID, err)
		return
	}

	payload, err := json.Marshal(result)
	if err != nil {
		log.Printf("mqtt: marshal result for %s: %v", deviceID, err)
		return
	}
	b.client.Publish(resultTopic(deviceID), qosAtLeastOnce, false, payload)
}

// handleReading parses and stores a telemetry message.
func (b *Broker) handleReading(_ paho.Client, msg paho.Message) {
	deviceID, ok := deviceIDFromTopic(msg.Topic())
	if !ok {
		log.Printf("mqtt: bad reading topic %q", msg.Topic())
		return
	}

	var payload struct {
		Metric     domain.Metric `json:"metric"`
		Value      float64       `json:"value"`
		Unit       string        `json:"unit"`
		BatchID    *uint         `json:"batch_id"`
		RecordedAt *time.Time    `json:"recorded_at"`
	}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("mqtt: bad reading payload from %s: %v", deviceID, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeout)
	defer cancel()

	b.touch(ctx, deviceID)

	reading := &domain.SensorReading{
		DeviceID: deviceID,
		BatchID:  payload.BatchID,
		Metric:   payload.Metric,
		Value:    payload.Value,
		Unit:     payload.Unit,
	}
	if payload.RecordedAt != nil {
		reading.RecordedAt = *payload.RecordedAt
	}
	if err := b.readings.Record(ctx, reading); err != nil {
		log.Printf("mqtt: record reading from %s: %v", deviceID, err)
	}
}

// touch best-effort updates a device's last-seen timestamp.
func (b *Broker) touch(ctx context.Context, deviceID string) {
	if err := b.devices.TouchLastSeen(ctx, deviceID); err != nil {
		log.Printf("mqtt: touch last_seen for %s: %v", deviceID, err)
	}
}
