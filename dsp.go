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
