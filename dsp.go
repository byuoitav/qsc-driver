package qsc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/fatih/color"
)

//DSP .
type DSP struct {
	Address string
}

const _kTimeoutInSeconds = 2.0

func (d *DSP) getConnection(ctx context.Context, port string) (*net.TCPConn, error) {
	addr, err := net.ResolveTCPAddr("tcp", d.Address+":"+port)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
	}
	return conn, err
}

//SendCommand sends a command to the DSP
func (d *DSP) SendCommand(ctx context.Context, request interface{}) ([]byte, error) {
	log.Printf(color.HiBlueString("[QSC-Communication] Sending a requst to %v", d.Address))
	toSend, err := json.Marshal(request)
	if err != nil {
		errmsg := fmt.Sprintf("[QSC-Communication] Invalid request, could not marshal: %v", err.Error())
		log.Printf(color.HiRedString(errmsg))
		return []byte{}, errors.New(errmsg)
	}

	conn, err := d.getConnection(ctx, "1710")
	if err != nil {
		return []byte{}, err
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	conn.SetReadDeadline(time.Now().Add(time.Duration(_kTimeoutInSeconds) * time.Second))
	msg, err := reader.ReadBytes('\x00')
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return []byte{}, err
	}

	//we can validate that the message is what we think it should be
	report := QSCStatusReport{}
	msg = bytes.Trim(msg, "\x00")

	err = json.Unmarshal(msg, &report)
	if err != nil {
		errmsg := fmt.Sprintf("[QSC-Communication] bad state recieved from device %v on connection: %s. Error: %v", d.Address, msg, err.Error())
		log.Printf(color.HiRedString(errmsg))
		return []byte{}, errors.New(errmsg)
	}

	//now we write our command
	conn.Write(toSend)
	conn.Write([]byte{0x00})

	conn.SetReadDeadline(time.Now().Add(time.Duration(_kTimeoutInSeconds) * time.Second))
	msg, err = reader.ReadBytes('\x00')
	if err != nil {
		log.Printf(color.HiRedString("Error: %v", err.Error()))
		return []byte{}, err
	}
	msg = bytes.Trim(msg, "\x00")
	log.Printf("%s", toSend)
	log.Printf("%s", msg)
	log.Printf(color.HiBlueString("[QSC-Communication] Done with request to %v.", d.Address))
	return msg, nil
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
