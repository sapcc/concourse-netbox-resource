package netbox

import (
	"fmt"
	"time"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
)

func getReferenceTime(input concourse.Input) (time.Time, error) {
	// If version.last_updated is provided, use it
	if input.Version.LastUpdated != "" {
		referenceTime, err := time.Parse(time.RFC3339, input.Version.LastUpdated)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid reference time format found in 'version.last_updated': %w", err)
		}
		return referenceTime.UTC(), nil
	}

	// If source.initial_lookback is provided, calculate time from now
	if input.Source.InitialLookback != "" {
		lookbackDuration, err := time.ParseDuration(input.Source.InitialLookback)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid lookback duration format (use Go duration format like 168h, 720h): %w", err)
		}
		return time.Now().UTC().Add(-lookbackDuration), nil
	}

	// Default to epoch 0 if neither is provided
	return time.Unix(0, 0).UTC(), nil
}

func getTimestamps(device any, input concourse.Input, fallbackReferenceTime *time.Time) (*time.Time, time.Time, error) {
	switch device := device.(type) {
	case netbox.DeviceWithConfigContext:
		lastUpdatedTime = device.LastUpdated.Get()
	case netbox.Interface:
		lastUpdatedTime = device.LastUpdated.Get()
	default:
		return &time.Time{}, time.Time{}, fmt.Errorf("unexpected device type in getTimestamps: %T", device)
	}

	// Get reference time from input, or use fallback if not provided
	if input.Version.LastUpdated != "" {
		referenceTime, err = getReferenceTime(input)
		if err != nil {
			return &time.Time{}, time.Time{}, fmt.Errorf("error parsing netbox lastupdated timestamp because of: %w", err)
		}
	} else if fallbackReferenceTime != nil {
		referenceTime = *fallbackReferenceTime
	} else {
		referenceTime = time.Time{}
	}

	return lastUpdatedTime, referenceTime, nil
}