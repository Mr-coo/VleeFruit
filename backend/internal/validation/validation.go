// Package validation holds reusable input-validation rules applied to
// untrusted data entering the system (REST bodies, MQTT payloads) before it
// reaches services or the database.
package validation

import (
	"errors"
	"math"
	"regexp"
	"unicode/utf8"
)

const (
	maxDeviceIDLen = 64
	maxUnitLen     = 16
)

// deviceIDPattern keeps device IDs safe to embed in object-storage keys and
// MQTT topic strings: no slashes, whitespace, or wildcard characters.
var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// DeviceID validates the public device identifier.
func DeviceID(s string) error {
	if s == "" {
		return errors.New("device_id is required")
	}
	if len(s) > maxDeviceIDLen {
		return errors.New("device_id too long")
	}
	if !deviceIDPattern.MatchString(s) {
		return errors.New("device_id must be 1-64 chars of letters, digits, '.', '_' or '-'")
	}
	return nil
}

// ReadingValue rejects non-finite sensor values (NaN, ±Inf) that would poison
// aggregates or JSON-encode into invalid output.
func ReadingValue(v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return errors.New("value must be a finite number")
	}
	return nil
}

// Unit validates an optional measurement unit string.
func Unit(s string) error {
	if utf8.RuneCountInString(s) > maxUnitLen {
		return errors.New("unit too long")
	}
	return nil
}
