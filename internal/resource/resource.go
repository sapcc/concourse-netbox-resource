package resource

type NetboxSource struct {
	NetboxUrl   string `json:"netbox-url"`
	NetboxToken string `json:"netbox-token"`
	APod        string `json:"apod"`
}
type NetboxVersion struct {
	Hash uint64 `json:"hash"`
}

type NetboxParams struct{}

type NetboxResource struct {
	source  *NetboxSource
	version *NetboxVersion
	params  *NetboxParams
}

func NewNetboxResource() *NetboxResource {
	return &NetboxResource{
		source:  &NetboxSource{},
		version: &NetboxVersion{},
		params:  &NetboxParams{},
	}
}

func (r *NetboxResource) Source() (source interface{}) {
	return r.source
}

func (r *NetboxResource) Version() (version interface{}) {
	return r.version
}

func (r *NetboxResource) Params() (params interface{}) {
	return r.params
}
