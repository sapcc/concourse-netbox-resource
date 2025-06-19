package filter

type NetboxObject struct {
	SiteName         []string        `json:"site_name,omitempty"`
	Tag              []string        `json:"tag,omitempty"`
	Role             []string        `json:"role,omitempty"`
	DeviceId         []int32         `json:"device_id,omitempty"`
	DeviceName       []string        `json:"device_name,omitempty"`
	DeviceType       []string        `json:"device_type,omitempty"`
	DeviceStatus     []string        `json:"device_status,omitempty"`
	ServerInterface  ServerInterface `json:"server_interface,omitempty"`
	GetConfigContext *bool           `json:"get_config_context,omitempty"`
}

type ServerInterface struct {
	InterfaceId   []int32  `json:"interface_id,omitempty"`
	InterfaceName []string `json:"interface_name,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
	MgmtOnly      *bool    `json:"mgmt_only,omitempty"`
	Connected     *bool    `json:"connected,omitempty"`
	Cabled        *bool    `json:"cabled,omitempty"`
	Type          []string `json:"type,omitempty"`
}
