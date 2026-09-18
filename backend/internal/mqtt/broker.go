package mqtt

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	"github.com/Mr-coo/VleeFruit/backend/internal/domain"
	"github.com/Mr-coo/VleeFruit/backend/internal/repository"
	"github.com/Mr-coo/VleeFruit/backend/internal/service"
	"github.com/Mr-coo/VleeFruit/backend/internal/validation"
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
	log           *slog.Logger
	maxImageBytes int64
}

func NewBroker(
	ingestion *service.IngestionService,
	readings *service.ReadingService,
	devices *repository.DeviceRepository,
) *Broker {
	return &Broker{
		ingestion: ingestion,
		readings:  readings,
		devices:   devices,
		log:       slog.Default().With("component", "mqtt"),
	}
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
		b.log.Warn("initial connect failed, will keep retrying", "error", token.Error())
	}
	return nil
}

// onConnect (re)subscribes on every successful connection.
func (b *Broker) onConnect(c paho.Client) {
	if token := c.Subscribe(topicImagesWildcard, qosAtLeastOnce, b.handleImage); token.Wait() && token.Error() != nil {
		b.log.Error("subscribe failed", "topic", topicImagesWildcard, "error", token.Error())
	}
	if token := c.Subscribe(topicReadingsWildcard, qosAtLeastOnce, b.handleReading); token.Wait() && token.Error() != nil {
		b.log.Error("subscribe failed", "topic", topicReadingsWildcard, "error", token.Error())
	}
	b.log.Info("connected and subscribed",
		"images_topic", topicImagesWildcard, "readings_topic", topicReadingsWildcard)
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
		b.log.Warn("bad image topic", "topic", msg.Topic())
		return
	}

	if b.maxImageBytes > 0 && int64(len(msg.Payload())) > b.maxImageBytes {
		b.log.Warn("image rejected: payload too large",
			"device_id", deviceID, "size_bytes", len(msg.Payload()), "limit_bytes", b.maxImageBytes)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeout)
	defer cancel()

	b.touch(ctx, deviceID)

	result, err := b.ingestion.IngestImage(ctx, deviceID, msg.Payload())
	if err != nil {
		b.log.Error("ingest image failed", "device_id", deviceID, "error", err)
		return
	}

	payload, err := json.Marshal(result)
	if err != nil {
		b.log.Error("marshal result failed", "device_id", deviceID, "error", err)
		return
	}
	b.client.Publish(resultTopic(deviceID), qosAtLeastOnce, false, payload)
	b.log.Info("image ingested and result published",
		"device_id", deviceID, "ripeness", result.Ripeness, "confidence", result.Confidence)
}

// handleReading parses and stores a telemetry message.
func (b *Broker) handleReading(_ paho.Client, msg paho.Message) {
	deviceID, ok := deviceIDFromTopic(msg.Topic())
	if !ok {
		b.log.Warn("bad reading topic", "topic", msg.Topic())
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
		b.log.Warn("bad reading payload", "device_id", deviceID, "error", err)
		return
	}
	if !payload.Metric.Valid() {
		b.log.Warn("reading rejected: invalid metric", "device_id", deviceID, "metric", payload.Metric)
		return
	}
	if err := validation.ReadingValue(payload.Value); err != nil {
		b.log.Warn("reading rejected: invalid value", "device_id", deviceID, "error", err)
		return
	}
	if err := validation.Unit(payload.Unit); err != nil {
		b.log.Warn("reading rejected: invalid unit", "device_id", deviceID, "error", err)
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
		b.log.Error("record reading failed", "device_id", deviceID, "error", err)
		return
	}
	b.log.Debug("reading recorded",
		"device_id", deviceID, "metric", payload.Metric, "value", payload.Value)
}

// touch best-effort updates a device's last-seen timestamp.
func (b *Broker) touch(ctx context.Context, deviceID string) {
	if err := b.devices.TouchLastSeen(ctx, deviceID); err != nil {
		b.log.Warn("touch last_seen failed", "device_id", deviceID, "error", err)
	}
}
