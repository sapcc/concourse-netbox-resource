package resource

import (
	"github.com/sapcc/go-netbox-go/dcim"
	"github.com/sapcc/go-netbox-go/models"
	fl "github.com/tbe/resource-framework/log"
)

func (r *NetboxResource) Check() (version interface{}, err error) {
	log := fl.NewDefaultLogger()
	log.SetLevel(0)
	nbc, err := dcim.New(r.source.NetboxUrl, r.source.NetboxToken, false)
	if err != nil {
		log.Error("error creating netbox client: %s", err)
		return nil, err
	}
	rackParams := models.ListRacksRequest{}
	rackParams.Q = r.source.APod
	racks, err := nbc.ListRacks(rackParams)
	if err != nil {
		log.Error("error listing racks: %s", err)
		return nil, err
	}
	if racks.Count != 1 {
		log.Error("wrong number of racks found: %d", racks.Count)
		return nil, err
	}

	params := models.ListDevicesRequest{}
	params.RackId = racks.Results[0].Id
	//TODO: make this dynamic
	params.DeviceTypeId = 132
	res, err := nbc.ListDevices(params)
	if err != nil {
		log.Error("error listing netbox devices: %s", err)
		return nil, err
	}
	for _, v := range res.Results {
		log.Warn("%v\n", v.Name)
	}
	return nil, nil
}
