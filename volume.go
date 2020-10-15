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

func (d *DSP) SetMute(ctx context.Context, block string, mute bool) error {

	//we generate our set status request, then we ship it out

	req := d.GetGenericSetStatusRequest(ctx)
	req.Params.Name = block
	if mute {
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

func (d *DSP) SetVolume(ctx context.Context, block string, volume int) error {

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

func (d *DSP) GetVolumes(ctx context.Context, blocks []string) (map[string]int, error) {
	toReturn := make(map[string]int)

	for i, block := range blocks {
		resp, err := d.GetControlStatus(ctx, block)
		if err != nil {
			log.Printf(color.HiRedString("There was an error: %v", err.Error()))
			return toReturn, err
		}

		log.Printf(color.HiBlueString("[QSC-Communication] Response received: %+v", resp))

		//get the volume out of the dsp and run it through our equation to reverse it
		found := false
		for _, res := range resp.Result {
			if res.Name == block {
				toReturn[blocks[i]] = d.DbToVolumeLevel(ctx, res.Value)
				found = true
				break
			}
		}
		if found {
			continue
		}

		return toReturn, errors.New("[QSC-Communication] No value returned with the name matching the requested state")
	}

	return toReturn, nil
}

func (d *DSP) GetMutes(ctx context.Context, blocks []string) (map[string]bool, error) {
	toReturn := make(map[string]bool)

	for i, block := range blocks {
		resp, err := d.GetControlStatus(ctx, block)
		if err != nil {
			log.Printf(color.HiRedString("There was an error: %v", err.Error()))
			return toReturn, err
		}

		//get the volume out of the dsp and run it through our equation to reverse it
		found := false
		for _, res := range resp.Result {
			if res.Name == block {
				if res.Value == 1.0 {
					toReturn[blocks[i]] = true
					found = true
				}
				if res.Value == 0.0 {
					toReturn[blocks[i]] = false
					found = true
				}
				break
			}
		}
		if found {
			continue
		}

		errmsg := "[QSC-Communication] No value returned with the name matching the requested state"
		log.Printf(color.HiRedString(errmsg))
		return toReturn, errors.New(errmsg)
	}

	return toReturn, nil
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
