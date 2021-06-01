package resource

import (
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
	"github.com/sapcc/go-netbox-go/dcim"
	"github.com/sapcc/go-netbox-go/models"
	fl "github.com/tbe/resource-framework/log"
)

type tmpHolder struct {
	Name        string
	Status      string
	LastUpdated string
}

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
	tmpArray := make([]tmpHolder, res.Count)
	for _, v := range res.Results {
		tmp := tmpHolder{
			Name:        v.Name,
			Status:      v.Status.Value,
			LastUpdated: v.LastUpdated,
		}
		log.Warn("%v\n", tmp)
		tmpArray = append(tmpArray, tmp)
	}
	hash, err := hashstructure.Hash(tmpArray, hashstructure.FormatV2, nil)
	if err != nil {
		log.Error("error generating hash: %s", err)
		return nil, err
	}
	log.Info("Hash: %d\n", hash)
	hashstr := strconv.FormatUint(hash, 10)
	if hashstr == r.version.Hash {
		return []NetboxVersion{}, nil
	} else {
		r.version.Hash = hashstr
		return []NetboxVersion{*r.version}, nil
	}
}
