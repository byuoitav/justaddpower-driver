package justaddpower

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/structs"
)

type JustAddPowerReciever struct {
	Address string
}

//JustAddPowerChannelResult type for result
type JustAddPowerChannelResult struct {
	Data string `json:"data"`
}

//JustAddPowerChannelIntResult type for result
type JustAddPowerChannelIntResult struct {
	Data int `json:"data"`
}

// JustAddPowerDetailsResult type for the hardware details stuff
type JustAddPowerDetailsResult struct {
	Data struct {
		Firmware struct {
			Date   string `json:"date"`
			Update struct {
				Eta      bool   `json:"eta"`
				Message  string `json:"message"`
				Progress bool   `json:"progress"`
				Result   bool   `json:"result"`
				Status   bool   `json:"status"`
			} `json:"update"`
			Version string `json:"version"`
		} `json:"firmware"`
		Model   string `json:"model"`
		Network struct {
			Ipaddress string `json:"ipaddress"`
			Mac       string `json:"mac"`
			Mtu       int    `json:"mtu"`
			Netmask   string `json:"netmask"`
			Speed     string `json:"speed"`
		} `json:"network"`
		Status string `json:"status"`
		Time   string `json:"time"`
		Uptime string `json:"uptime"`
	} `json:"data"`
}

func justAddPowerRequest(url string, body string, method string) ([]byte, error) {

	var netRequest, err = http.NewRequest(method, url, bytes.NewReader([]byte(body)))

	if err != nil {
		return nil, fmt.Errorf("Error when creating new just add power netrequest")
	}

	var netClient = http.Client{
		Timeout: time.Second * 10,
	}

	response, err := netClient.Do(netRequest)

	if err != nil {
		return nil, fmt.Errorf("Error when posting to Just add power device")
	}

	defer response.Body.Close()

	bytes, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, fmt.Errorf("Error when reading Just add power device response body")
	}

	if response.StatusCode/100 != 2 {
		return bytes, fmt.Errorf("Just add power device did not return HTTP OK")
	}

	return bytes, nil
}
func checkTransmitterChannel(address string) {
	channel, err := getTransmissionChannelforAddress(address)

	ipAddress, err2 := net.ResolveIPAddr("ip", address)
	ipAddress.IP = ipAddress.IP.To4()

	if err == nil && err2 == nil {
		if string(ipAddress.IP[3]) == channel {
			//we're good
			return
		}
	}
	setTransmitterChannelForAddress(address)
}
func setTransmitterChannelForAddress(transmitter string) (string, error) {
	ipAddress, err := net.ResolveIPAddr("ip", transmitter)
	ipAddress.IP = ipAddress.IP.To4()

	if err != nil {
		return "", fmt.Errorf("Error when resolving IP Address [%s]: %w", transmitter, err)
	}

	log.L.Debugf("Setting transmitter ipaddr %v", ipAddress)

	channel := fmt.Sprintf("%v", ipAddress.IP[3])

	log.L.Debugf("Setting transmitter channel %+v", channel)

	result, er := justAddPowerRequest(fmt.Sprintf("http://%s/cgi-bin/api/command/channel", transmitter), channel, "POST")

	log.L.Debugf("Result %v", result)

	if er != nil {
		return "", fmt.Errorf("Error when making request: %w", er)
	}

	return "ok", nil
}

func getTransmissionChannelforAddress(address string) (string, error) {
	log.L.Debugf("Getting transmitter channel for address %v", address)

	ipAddress, err := net.ResolveIPAddr("ip", address)
	ipAddress.IP = ipAddress.IP.To4()

	log.L.Debugf("%+v", ipAddress.IP)

	if err != nil {
		return "", fmt.Errorf("Error when resolving IP Address [%s]", address)
	}

	result, errrrrr := justAddPowerRequest(fmt.Sprintf("http://%s/cgi-bin/api/details/channel", address), "", "GET")

	if errrrrr != nil {
		log.L.Debugf("%v", err)
		return "", fmt.Errorf("Error when making request: %w", errrrrr)
	}

	var jsonResult JustAddPowerChannelIntResult
	gerr := json.Unmarshal(result, &jsonResult)
	if gerr != nil {
		log.L.Debugf("%v", err)
		return "", fmt.Errorf("Error when unmarshaling response: %w", gerr)
	}
	log.L.Debugf("Result %s %v", result, jsonResult)
	log.L.Debugf("len of IP %v", len(ipAddress.IP))

	transmissionChannel := fmt.Sprintf("%v.%v.%v.%v",
		ipAddress.IP[0], ipAddress.IP[1], ipAddress.IP[2], jsonResult.Data)

	return transmissionChannel, nil
}

// GetAudioVideoInputs returns the current input
func (j *JustAddPowerReciever) GetAudioVideoInputs(ctx context.Context) (map[string]string, error) {
	toReturn := make(map[string]string)

	ipAddress, err := net.ResolveIPAddr("ip", j.Address)
	ipAddress.IP = ipAddress.IP.To4()

	log.L.Debugf("%+v", ipAddress.IP)

	if err != nil {
		return toReturn, fmt.Errorf("Error when resolving IP Address [%s]: %w", j.Address, err)
	}

	result, err := justAddPowerRequest(fmt.Sprintf("http://%s/cgi-bin/api/details/channel", j.Address), "", "GET")

	if err != nil {
		log.L.Debugf("%v", err)
		return toReturn, fmt.Errorf("error when making request: %w", err)
	}

	var jsonResult JustAddPowerChannelIntResult
	gerr := json.Unmarshal(result, &jsonResult)
	if gerr != nil {
		log.L.Debugf("%v", gerr)
		return toReturn, fmt.Errorf("error when unmarshaling response: %w", gerr)
	}

	log.L.Debugf("Result %s %v", result, jsonResult)
	log.L.Debugf("len of IP %v", len(ipAddress.IP))

	transmissionChannel := fmt.Sprintf("%v.%v.%v.%v",
		ipAddress.IP[0], ipAddress.IP[1], ipAddress.IP[2], jsonResult.Data)

	toReturn[""] = transmissionChannel
	return toReturn, nil
}

// SwitchInput changes the input on the given output to input (Just add power transmitter - ipaddr)
// We don't need the output necessarily because the reciever is the output
func (j *JustAddPowerReciever) SetInput(ctx context.Context, output, input string) error {
	log.L.Debugf("Setting receiver to transmitter")

	go checkTransmitterChannel(input)

	log.L.Debugf("Routing %v to %s", j.Address, input)

	ipAddress, err := net.ResolveIPAddr("ip", input)
	ipAddress.IP = ipAddress.IP.To4()

	if err != nil {
		return fmt.Errorf("Error when resolving IP Address [%s]: %w", input, err)
	}

	channel := fmt.Sprintf("%v", ipAddress.IP[3])

	log.L.Debugf("Channel %v", channel)

	result, errrr := justAddPowerRequest(fmt.Sprintf("http://%s/cgi-bin/api/command/channel", j.Address), channel, "POST")

	if errrr != nil {
		return fmt.Errorf("Error when making request: %w", errrr)
	}

	var jsonResult JustAddPowerChannelResult
	err = json.Unmarshal(result, &jsonResult)

	log.L.Debugf("Result %v", jsonResult)

	if err != nil {
		return fmt.Errorf("Error when unpacking json")
	}
	return nil
}

// GetHardwareInfo returns a hardware info struct
func (j *JustAddPowerReciever) GetHardwareInfo(ctx context.Context) (structs.HardwareInfo, error) {
	var details structs.HardwareInfo

	addr, e := net.LookupAddr(j.Address)
	if e != nil {
		details.Hostname = j.Address
	} else {
		details.Hostname = strings.Trim(addr[0], ".")
	}

	//Send the request to the Just Add Power API
	result, err := justAddPowerRequest(fmt.Sprintf("http://%s/cgi-bin/api/details/device", j.Address), "", "GET")
	if err != nil {
		return details, fmt.Errorf("Error occured whilemaking request:%w", err)
	}

	//jsonResult is the response back from the API, it has all the information
	var jsonResult JustAddPowerDetailsResult
	gerr := json.Unmarshal(result, &jsonResult)
	if gerr != nil {
		return details, fmt.Errorf("Error occured whilemaking request:%w", err)
	}

	details.ModelName = jsonResult.Data.Model                  //Model of the device
	details.FirmwareVersion = jsonResult.Data.Firmware.Version //Version of firmware on the device
	details.BuildDate = jsonResult.Data.Firmware.Date          //The Date the firmware was released
	details.PowerStatus = jsonResult.Data.Uptime               //Reports on how long the device has been booted

	// Get the Network info stuff
	details.NetworkInfo.IPAddress = jsonResult.Data.Network.Ipaddress
	details.NetworkInfo.MACAddress = jsonResult.Data.Network.Mac

	return details, nil
}

//GetInfo .
func (j *JustAddPowerReciever) GetInfo(ctx context.Context) (interface{}, error) {
	var info interface{}
	return info, fmt.Errorf("not currently implemented")
}
