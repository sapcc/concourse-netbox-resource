package concourse

import (
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
)

type Input struct {
	Source  Source  `json:"source"`
	Version Version `json:"version,omitempty"`
}

type Output struct {
	Version  Version    `json:"version,omitempty"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

type Source struct {
	Url    string              `json:"url"`
	Token  string              `json:"token,omitempty"`
	Filter filter.NetboxObject `json:"filter,omitempty"`
}

type Version struct {
	Id                  string `json:"id"`
	LastUpdated         string `json:"last_updated"`
	ObjectType          string `json:"object_type"`
	DeviceId            string `json:"device_id,omitempty"`
	DeviceName          string `json:"device_name"`
	DeviceRole          string `json:"device_role"`
	DeviceApiUrl        string `json:"device_api_url,omitempty"`
	DeviceDisplayUrl    string `json:"device_display_url,omitempty"`
	ConfigContext       string `json:"config_context,omitempty"`
	InterfaceName       string `json:"interface_name,omitempty"`
	InterfaceType       string `json:"interface_type,omitempty"`
	InterfaceApiUrl     string `json:"interface_api_url,omitempty"`
	InterfaceDisplayUrl string `json:"interface_display_url,omitempty"`
}

type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
