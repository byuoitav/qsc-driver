package qsc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/fatih/color"
)

//SetMutedByBlock sets the mute
func (d *DSP) SetMutedByBlock(ctx context.Context, block string, muted bool) error {
	//we generate our set status request, then we ship it out

	block = block + "Mute"
	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block
	if muted {
		req.Params.Value = 1
	} else {
		req.Params.Value = 0
	}

	resp, err := d.SendCommand(ctx, req)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return err
	}

	//we need to unmarshal our response, parse it for the value we care about, then role with it from there
	val := QSCSetStatusResponse{}
	err = json.Unmarshal(resp, &val)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return err
	}

	//otherwise we check to see what the value is set to
	if val.Result.Name != block {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", block, val.Result.Name)
		log.Printf(color.HiRedString(errmsg))
		return errors.New(errmsg)
	}

	if val.Result.Value == 1.0 {
		return nil
	}
	if val.Result.Value == 0.0 {
		return nil
	}
	errmsg := fmt.Sprintf("[QSC-Communication] Invalid response received: %v", val.Result)
	log.Printf(color.HiRedString(errmsg))
	return errors.New(errmsg)
}

//SetVolumeByBlock sets the volume
func (d *DSP) SetVolumeByBlock(ctx context.Context, block string, volume int) error {
	block = block + "Gain"
	log.Printf("got: %v", volume)
	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block

	if volume == 0 {
		req.Params.Value = -100
	} else {
		//do the logrithmic magic
		req.Params.Value = d.VolToDb(ctx, volume)
	}
	log.Printf("sending: %v", req.Params.Value)

	resp, err := d.SendCommand(ctx, req)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return err
	}

	//we need to unmarshal our response, parse it for the value we care about, then role with it from there
	val := QSCSetStatusResponse{}
	err = json.Unmarshal(resp, &val)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return err
	}
	if val.Result.Name != block {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", block, val.Result.Name)
		log.Printf(color.HiRedString(errmsg))
		return errors.New(errmsg)
	}

	return nil
}

func (d *DSP) DbToVolumeLevel(ctx context.Context, level float64) int {
	return int(math.Pow(10, (level/20)) * 100)
}

func (d *DSP) VolToDb(ctx context.Context, level int) float64 {
	return math.Log10(float64(level)/100) * 20
}

//GetVolumeByBlock gets the volume
func (d *DSP) GetVolumeByBlock(ctx context.Context, block string) (int, error) {

	block = block + "Gain"
	resp, err := d.GetControlStatus(ctx, block)
	if err != nil {
		log.Printf(color.HiRedString("There was an error: %v", err.Error()))
		//LOOK AT THIS LATER
		return 0, err
	}

	log.Printf(color.HiBlueString("[QSC-Communication] Response received: %+v", resp))

	//get the volume out of the dsp and run it through our equation to reverse it
	for _, res := range resp.Result {
		if res.Name == block {
			//this uses common and we need to make it not somehow
			return d.DbToVolumeLevel(ctx, res.Value), nil
		}
	}

	//this uses common and we need to make it not somehow
	return 0, errors.New("[QSC-Communication] No value returned with the name matching the requested state")
}

//GetMutedByBlock gets the mute status
func (d *DSP) GetMutedByBlock(ctx context.Context, block string) (bool, error) {
	block = block + "Mute"
	resp, err := d.GetControlStatus(ctx, block)
	if err != nil {
		log.Printf(color.HiRedString("There was an error: %v", err.Error()))
		//LOOK AT THIS LATER
		return true, err
	}

	//get the volume out of the dsp and run it through our equation to reverse it
	for _, res := range resp.Result {
		if res.Name == block {
			if res.Value == 1.0 {
				return true, nil
			}
			if res.Value == 0.0 {
				return false, nil
			}
		}
	}
	errmsg := "[QSC-Communication] No value returned with the name matching the requested state"
	log.Printf(color.HiRedString(errmsg))
	//LOOK AT THIS LATER
	return false, errors.New(errmsg)
}

func (d *DSP) GetControlStatus(ctx context.Context, name string) (QSCGetStatusResponse, error) {
	req := d.GetGenericGetStatusRequest(ctx)
	req.Params = append(req.Params, name)

	toReturn := QSCGetStatusResponse{}

	resp, err := d.SendCommand(ctx, req)
	if err != nil {
		log.Printf(color.HiRedString(err.Error()))
		return toReturn, err
	}

	err = json.Unmarshal(resp, &toReturn)
	if err != nil {
		log.Printf(color.HiRedString(err.Error()))
	}

	return toReturn, err
}

func (d *DSP) SetControlStatus(ctx context.Context, name, value string) (QSCSetStatusResponse, error) {
	var err error
	req := d.GetGenericSetStatusRequest(ctx)
	val := QSCSetStatusResponse{}

	req.Params.Name = name
	req.Params.Value, err = strconv.ParseFloat(value, 64)
	if err != nil {
		return val, errors.New("Invalid value, must be a float")
	}
	log.Printf("sending: %v:%v to %v", req.Params.Name, req.Params.Value, d.Address)

	resp, err := d.SendCommand(ctx, req)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return val, err
	}

	//we need to unmarshal our response, parse it for the value we care about, then role with it from there
	err = json.Unmarshal(resp, &val)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return val, err
	}
	if val.Result.Name != name {
		errmsg := fmt.Sprintf("Invalid response, the name recieved does not match the name sent %v/%v", name, val.Result.Name)
		log.Printf(color.HiRedString(errmsg))
		return val, errors.New(errmsg)
	}

	return val, nil
}
