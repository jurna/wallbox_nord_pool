package wallbox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DeviceId string `yaml:"device-id"`
}

type Wallbox struct {
	token    string
	deviceId string
}

const tokenFile = "user_token.json"

var (
	errTokenFileDoesNotExist = errors.New("token file does not exist")
)

type ChargerStatus string

var DefaultClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

const (
	Unknown       ChargerStatus = "Unknown"
	Waiting                     = "Waiting"
	Charging                    = "Charging"
	Ready                       = "Ready"
	Paused                      = "Paused"
	Scheduled                   = "Scheduled"
	Discharging                 = "Discharging"
	Error                       = "Error"
	Disconnected                = "Disconnected"
	Locked                      = "Locked"
	LockedWaiting               = "LockedWaiting"
	Updating                    = "Updating"
)

var intToStatusMap = map[int]ChargerStatus{
	164: Waiting,
	180: Waiting,
	181: Waiting,
	183: Waiting,
	184: Waiting,
	185: Waiting,
	186: Waiting,
	187: Waiting,
	188: Waiting,
	189: Waiting,
	193: Charging,
	194: Charging,
	195: Charging,
	161: Ready,
	162: Ready,
	178: Paused,
	182: Paused,
	177: Scheduled,
	179: Scheduled,
	196: Discharging,
	14:  Error,
	15:  Error,
	0:   Disconnected,
	163: Disconnected,
	209: Locked,
	210: LockedWaiting,
	165: Locked,
	166: Updating,
}

type UserToken struct {
	Jwt string `json:"jwt"`
	Ttl int64  `json:"ttl"`
}
type ChargerData struct {
	Data struct {
		ChargerData struct {
			Id     int `json:"id"`
			Status int `json:"status"`
			Locked int `json:"locked"`
		} `json:"chargerData"`
	} `json:"data"`
}

type ChargerAction struct {
	Locked int `json:"locked"`
}

type RemoteAction struct {
	Action int `json:"action"`
}

type ChargerConfig struct {
	EnergyCost float64 `json:"energyCost,omitempty"`
}

func NewWallbox(config Config) (wallbox Wallbox, err error) {
	token, err := getToken(config.Username, config.Password)
	if err != nil {
		return
	}
	return Wallbox{token, config.DeviceId}, err
}

func getToken(username string, password string) (token string, err error) {
	userToken, err := readToken()
	if err != nil {
		if !errors.Is(err, errTokenFileDoesNotExist) {
			return
		}
		userToken, err = getNewToken(username, password)
		if err != nil {
			return
		}
	}
	return userToken.Jwt, err
}

func (wallbox Wallbox) GetStatus() (status ChargerStatus, err error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.wall-box.com/v2/charger/%s", wallbox.deviceId), nil)
	if err != nil {
		return
	}
	wallbox.addHeaders(req)
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = handleHttpError(resp)
		return
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var chargerData = ChargerData{}
	err = json.Unmarshal(tokenBytes, &chargerData)
	if err != nil {
		return
	}
	status = mapToStatus(chargerData.Data.ChargerData.Status)
	return
}

func (wallbox Wallbox) Unlock() (err error) {
	marshallBytes, err := json.Marshal(ChargerAction{Locked: 0})
	if err != nil {
		return
	}
	body := bytes.NewReader(marshallBytes)
	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.wall-box.com/v2/charger/%s", wallbox.deviceId), body)
	if err != nil {
		return
	}
	wallbox.addHeaders(req)
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = handleHttpError(resp)
		return
	}

	_, err = io.ReadAll(resp.Body)
	return
}

func (wallbox Wallbox) SetEnergyCost(cost float64) (err error) {
	marshallBytes, err := json.Marshal(ChargerConfig{EnergyCost: cost})
	if err != nil {
		return
	}
	body := bytes.NewReader(marshallBytes)
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.wall-box.com/chargers/config/%s", wallbox.deviceId), body)
	if err != nil {
		return
	}
	wallbox.addHeaders(req)
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = handleHttpError(resp)
		return
	}

	_, err = io.ReadAll(resp.Body)
	return
}

func (wallbox Wallbox) PauseCharging() (err error) {
	return wallbox.remoteAction(RemoteAction{Action: 2})
}

func (wallbox Wallbox) ResumeCharging() (err error) {
	return wallbox.remoteAction(RemoteAction{Action: 1})
}

func (wallbox Wallbox) remoteAction(action RemoteAction) (err error) {
	marshallBytes, err := json.Marshal(action)
	if err != nil {
		return
	}
	body := bytes.NewReader(marshallBytes)
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.wall-box.com/v3/chargers/%s/remote-action", wallbox.deviceId), body)
	if err != nil {
		return
	}
	wallbox.addHeaders(req)
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = handleHttpError(resp)
		return
	}

	_, err = io.ReadAll(resp.Body)
	return
}

func (wallbox Wallbox) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", wallbox.token))
	req.Header.Set("Content-Type", "application/json")
}

func mapToStatus(status int) (chargerStatus ChargerStatus) {
	val, ok := intToStatusMap[status]
	if ok {
		return val
	} else {
		return Unknown
	}
}

func getNewToken(username string, password string) (token UserToken, err error) {
	req, err := http.NewRequest("GET", "https://api.wall-box.com/auth/token/user", nil)
	if err != nil {
		return
	}
	req.SetBasicAuth(username, password)
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = handleHttpError(resp)
		return
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = writeToken(tokenBytes)
	if err != nil {
		return
	}
	err = json.Unmarshal(tokenBytes, &token)
	return
}

func handleHttpError(resp *http.Response) error {
	tokenBytes, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("invalid response %d %s", resp.StatusCode, tokenBytes)
}

func writeToken(token []byte) (err error) {
	f, err := os.Create(tokenFile)
	if err != nil {
		return
	}
	_, err = f.Write(token)
	return err
}

func readToken() (token UserToken, err error) {
	f, err := os.Open(tokenFile)
	if err != nil {
		return token, fmt.Errorf("%s - %w", tokenFile, errTokenFileDoesNotExist)
	}
	tokenBytes, err := io.ReadAll(f)
	if err != nil {
		_ = os.Remove(f.Name())
		return
	}
	err = json.Unmarshal(tokenBytes, &token)
	if err != nil {
		return
	}
	ttlTime := time.Unix(token.Ttl, 0)
	if ttlTime.Before(time.Now()) {
		_ = os.Remove(f.Name())
		return UserToken{}, fmt.Errorf("expired token. Ttl time %s - %w", ttlTime, errTokenFileDoesNotExist)
	}
	return
}
