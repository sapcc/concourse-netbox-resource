package helper

import (
	"fmt"
	"os"
)

var (
	ConcourseOutputPath   string = "/tmp/build/get"
	ConcourseInputPath    string = "/tmp/build/put"
	ConcourseSourceConfig string = `
		{
			"source": {
				"url": "https://netbox.example.local",
				"token": "your-api-token",
				"filter": {
					"site_name": [
						"site-a",
						"site-b",
						"site-c"
					],
					"tag": [
						"tag1",
						"tag2"
					],
					"role": [
						"server"
					],
					"device_id": [
						123,
						456
					],
					"device_name": [
						"srv-"
					],
					"device_type": [
						"vendor model"
					],
					"device_status": [
						"active"
					],
					"server_interface": {
						"enabled": true,
						"interface_id": [
							789,
							101
						],
						"interface_name": [
							"eth0"
						],
						"mgmt_only": false,
						"connected": true,
						"cabled": true,
						"type": [
							"virtual"
						]
					}
				}
			}
		}
	`
	ConcourseInvalidSourceConfig string = `
		{
			"source": {
				"url": "",
				"token": "your-api-token"
			}
		}
	`
	ConcourseVersion string = `
		{
			"source": {
				"url": "https://netbox.example.local",
				"token": "your-api-token"
			},
			"version": {
				"id": "456",
				"last_updated": "2025-06-23T15:16:56Z",
				"object_type": "interfaces",
				"device_id": "123",
				"device_name": "server01",
				"device_role": "server",
				"interface_name": "vmk1",
				"interface_type": "Virtual"
			}
		}
	`
	NetBoxConfigContextData = map[string]any{
		"management_ip": "192.168.1.100",
		"location":      "rack-1",
		"environment":   "production",
	}
	NetBoxInvalidConfigContextData = map[string]any{
		"management_ip": "192.168.1",
	}
	DeviceId         int32  = 123
	DeviceName       string = "test-device"
	DeviceApiUrl     string = "http://netbox.example.local/api/dcim/devices"
	DeviceDisplayUrl string = "http://netbox.example.local/dcim/devices"
	DeviceRoleId     int32  = 8
	DeviceRoleSlug   string = "server"
)

func EnsureFolder(path string) error {
	var err error

	if err = os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}
	return nil
}
