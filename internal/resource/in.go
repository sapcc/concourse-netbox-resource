package resource

func (r *NetboxResource) In(_ string) (version interface{}, metadata []interface{}, err error) {
	return r.version, nil, nil
}
