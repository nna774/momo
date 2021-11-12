package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/nna774/momochi/momochi"
)

var (
	switchbotAPIHost = "https://api.switch-bot.com"
	apiKey           = os.Getenv("SWITCHBOT_API_KEY")
	momochiEndpoint  = os.Getenv("MOMOCHI_ENDPOINT")
	meterID          = os.Getenv("METER_ID")
)

type DeviceType int

const (
	TypeUnknown DeviceType = iota
	TypeHub
	TypeBot
	TypeMeter
)

func (dt DeviceType) String() string {
	switch dt {
	case TypeMeter:
		return "Meter"
	default:
		return "unknown"
	}
}

func (dt *DeviceType) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return fmt.Errorf("unknown valude: %v", string(b))
	}
	value := string(b[1 : len(b)-1])
	switch value {
	case TypeBot.String():
		*dt = TypeBot
		return nil
	case TypeMeter.String():
		*dt = TypeMeter
		return nil
	default:
		return fmt.Errorf("undefined DeviceType: %v", value)
	}
}

type base struct {
	DeviceID    string     `json:"deviceId"`
	DeviceType  DeviceType `json:"deviceType"`
	HubDeviceID string     `json:"hubDeviceId"`
}

type Meter struct {
	base
	Humidity    int     `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

type SwitchBotAPIResponse struct {
	StatuCode int             `json:"statusCode"`
	Message   string          `json:"message"`
	Body      json.RawMessage `json:"body"`
}

func status(deviceID string) (*Meter, error) {
	uri := switchbotAPIHost + "/v1.0/devices/" + deviceID + "/status"
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json; charset=utf8")
	req.Header.Add("Authorization", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r SwitchBotAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, err
	}
	if r.StatuCode != 100 {
		return nil, fmt.Errorf("request failed. code: %v, message: %v", r.StatuCode, r.Message)
	}
	var m Meter
	err = json.Unmarshal(r.Body, &m)
	return &m, err
}

func post(m *Meter) error {
	mmc := momochi.NewClient(momochiEndpoint)
	res, err := mmc.PostTemp(momochi.NewTemp(float32(m.Temperature), float32(m.Humidity)))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	d, _ := ioutil.ReadAll(res.Body)
	fmt.Printf("res: %v", string(d))
	return nil
}

func HandleRequest() error {
	if apiKey == "" {
		return fmt.Errorf("need SWITCHBOT_API_KEY")
	}
	res, err := status(meterID)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", res)
	return post(res)
}

func main() {
	lambda.Start(HandleRequest)
}
