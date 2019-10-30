package qsc

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/common/structs"
	"github.com/fatih/color"
)

// GetDetails is all the juicy details about the QSC that everyone is DYING to know about
func (d *DSP) GetDetails(ctx context.Context) (structs.HardwareInfo, *nerr.E) {

	// toReturn is the struct of Hardware info
	var details structs.HardwareInfo

	// get the hostname
	addr, e := net.LookupAddr(d.Address)
	if e != nil {
		details.Hostname = d.Address
	} else {
		details.Hostname = strings.Trim(addr[0], ".")
	}

	resp, err := d.GetStatus(ctx)
	if err != nil {
		return details, nerr.Translate(err).Addf("There was an error getting the status")
	}

	log.L.Infof("response: %v", resp)
	details.ModelName = resp.Result.Platform
	details.PowerStatus = resp.Result.State

	details.NetworkInfo = structs.NetworkInfo{
		IPAddress: d.Address,
	}

	return details, nil
}

// GetStatus will be getting responses for us I hope...
func (d *DSP) GetStatus(ctx context.Context) (QSCStatusGetResponse, error) {
	req := d.GetGenericStatusGetRequest(ctx)

	log.L.Infof("In GetStatus...")
	toReturn := QSCStatusGetResponse{}

	resp, err := d.SendCommand(ctx, req)
	if err != nil {
		log.L.Info(color.HiRedString(err.Error()))
		return toReturn, err
	}

	err = json.Unmarshal(resp, &toReturn)
	if err != nil {
		log.L.Infof(color.HiRedString(err.Error()))
	}

	return toReturn, err
}
