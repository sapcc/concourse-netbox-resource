package netbox

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

var (
	currentTime time.Time
	updatedTime netbox.NullableTime
	role        netbox.BriefDeviceRole
	device      *netbox.DeviceWithConfigContext = netbox.NewDeviceWithConfigContextWithDefaults()
)

func TestConfigContextParsing(t *testing.T) {
	currentTime = time.Now()
	referenceTime = currentTime.AddDate(-10, 0, 0)
	updatedTime.Set(&currentTime)

	role.SetId(helper.DeviceRoleId)
	role.SetSlug(helper.DeviceRoleSlug)

	device.Id = helper.DeviceId
	device.Url = helper.DeviceApiUrl + fmt.Sprintf("/%d/", device.Id)
	device.Display = helper.DeviceName
	displayUrl := helper.DeviceDisplayUrl + fmt.Sprintf("/%d/", device.Id)
	device.DisplayUrl = &displayUrl
	device.LastUpdated = updatedTime
	device.Role = role
	device.SetConfigContext(helper.NetBoxConfigContextData)

	// Test cases
	configContextEnabled := true
	configContextDisabled := false

	tests := []struct {
		name                  string
		configContext         *bool
		expectedConfigContext *map[string]any
		expectConfigEmpty     bool
		expectValidJSON       bool
		wantErr               bool
	}{
		{"configContextDisabledNil", nil, nil, true, false, false},
		{"configContextExplicitlyEnabled", &configContextEnabled, &helper.NetBoxConfigContextData, false, true, false},
		{"configContextInvalid", &configContextEnabled, &helper.NetBoxInvalidConfigContextData, false, true, true},
		{"configContextExplicitlyDisabled", &configContextDisabled, nil, true, false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset global output variable before each test
			output = make([]concourse.Version, 0)

			input := concourse.Input{
				Source: concourse.Source{
					Filter: filter.NetboxObject{
						GetConfigContext: test.configContext,
					},
				},
				Version: concourse.Version{
					LastUpdated: referenceTime.Format(time.RFC3339),
				},
			}

			currentDevice := *device
			result, err := populateDeviceDetails(helper.DeviceName, input, currentDevice)
			if err != nil {
				t.Fatalf("Error in populateDeviceDetails: %v", err)
			}

			if len(result) == 0 {
				t.Errorf("Expected at least one output item, got none")
			}

			configContextResult := result[0].ConfigContext

			if !test.expectConfigEmpty && len(configContextResult) == 0 && !test.wantErr {
				t.Errorf("Expected config context data, got empty string")
			}

			if test.expectConfigEmpty && len(configContextResult) != 0 && !test.wantErr {
				t.Errorf("Expected empty config context, got: %s", configContextResult)
			}

			if test.expectValidJSON && !test.expectConfigEmpty && len(configContextResult) != 0 && !test.wantErr {
				var (
					expectedConfigContext map[string]any
					parsedContext         map[string]any
				)

				if err := json.Unmarshal([]byte(configContextResult), &parsedContext); err != nil {
					t.Fatalf("failed to decode configContextResult: %v", err)
				}

				expectedConfigContextBytes, err := json.Marshal(test.expectedConfigContext)
				if err != nil {
					t.Fatalf("failed to marshal test.expectedConfigContext: %v", err)
				}
				if err := json.Unmarshal(expectedConfigContextBytes, &expectedConfigContext); err != nil {
					t.Fatalf("failed to unmarshal expectedConfigContextBytes: %v", err)
				}

				if parsedContext["management_ip"] != expectedConfigContext["management_ip"] {
					t.Errorf("Expected management_ip to be '%s', got: %v",
						expectedConfigContext["management_ip"], parsedContext["management_ip"])
				}
			}
		})
	}
}
