package resource

type NetboxSource struct {
	NetboxUrl   string `json:"netbox-url"`
	NetboxToken string `json:"netbox-token"`
	APod        string `json:"apod"`
}
type NetboxVersion struct {
	Number int `json:"number"`
}

type NetboxResource struct {
	source  *NetboxSource
	version *NetboxVersion
}

func NewNetboxResource() *NetboxResource {
	return &NetboxResource{
		source:  &NetboxSource{},
		version: &NetboxVersion{},
	}
}

func (r *NetboxResource) Source() (source interface{}) {
	return r.source
}

func (r *NetboxResource) Version() (version interface{}) {
	return r.version
}
