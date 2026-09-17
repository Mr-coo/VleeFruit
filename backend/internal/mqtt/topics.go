package mqtt

import (
	"fmt"
	"strings"
)

// Topic layout (device is scoped to its own subtree by broker ACLs):
//
//	devices/{id}/images   device -> backend  (raw image bytes)
//	devices/{id}/readings device -> backend  (JSON telemetry)
//	devices/{id}/results  backend -> device  (JSON detection result)
const (
	topicImagesWildcard   = "devices/+/images"
	topicReadingsWildcard = "devices/+/readings"
)

// resultTopic returns the results topic for a specific device.
func resultTopic(deviceID string) string {
	return fmt.Sprintf("devices/%s/results", deviceID)
}

// deviceIDFromTopic extracts {id} from "devices/{id}/<suffix>".
func deviceIDFromTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" {
		return "", false
	}
	return parts[1], parts[1] != ""
}
