package support

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/ably/ably-go"
)

type testAppNamespace struct {
	ID        string `json:"id"`
	Persisted bool   `json:"persisted"`
}

type testAppKey struct {
	ID         string `json:"id,omitempty"`
	Value      string `json:"value,omitempty"`
	Capability string `json:"capability,omitempty"`
}

type testAppPresenceConfig struct {
	ClientID   string `json:"clientId"`
	ClientData string `json:"clientData"`
}

type testAppChannels struct {
	Name     string                  `json:"name"`
	Presence []testAppPresenceConfig `json:"presence"`
}

type testAppConfig struct {
	AppID      string             `json:"appId,omitempty"`
	Keys       []testAppKey       `json:"keys"`
	Namespaces []testAppNamespace `json:"namespaces"`
	Channels   []testAppChannels  `json:"channels"`
}

type TestApp struct {
	ClientOptions *ably.ClientOptions
	Config        testAppConfig
	HttpClient    *http.Client
}

func (t *TestApp) AppKeyValue() string {
	return t.Config.Keys[0].Value
}

func (t *TestApp) AppKeyId() string {
	return fmt.Sprintf("%s.%s", t.Config.AppID, t.Config.Keys[0].ID)
}

func (t *TestApp) ok(status int) bool {
	return status == http.StatusOK || status == http.StatusCreated || status == http.StatusNoContent
}

func (t *TestApp) populate(res *http.Response) error {
	if err := json.NewDecoder(res.Body).Decode(&t.Config); err != nil {
		return err
	}

	t.ClientOptions.ApiKey = t.AppKeyId() + ":" + t.AppKeyValue()

	return nil
}

func (t *TestApp) Create() (*http.Response, error) {
	buf, err := json.Marshal(t.Config)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", t.ClientOptions.RestEndpoint+"/apps", bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := t.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if !t.ok(res.StatusCode) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, fmt.Errorf("Unexpected status code %d. Body couldn't be parsed: %s", res.StatusCode, err)
		} else {
			return res, fmt.Errorf("Unexpected status code %d: %s", res.StatusCode, body)
		}
	}

	err = t.populate(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (t *TestApp) Delete() (*http.Response, error) {
	req, err := http.NewRequest("DELETE", t.ClientOptions.RestEndpoint+"/apps/"+t.Config.AppID, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(t.AppKeyId(), t.AppKeyValue())
	res, err := t.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if !t.ok(res.StatusCode) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return res, fmt.Errorf("Unexpected status code %d. Body couldn't be parsed: %s", res.StatusCode, err)
		} else {
			return res, fmt.Errorf("Unexpected status code %d: %s", res.StatusCode, body)
		}
	}

	return res, nil
}

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, time.Duration(10*time.Second))
}

func NewTestApp() *TestApp {
	transport := http.Transport{
		Dial: dialTimeout,
	}

	client := http.Client{
		Transport: &transport,
	}

	return &TestApp{
		HttpClient: &client,
		ClientOptions: &ably.ClientOptions{
			RealtimeEndpoint: "wss://sandbox-realtime.ably.io:443",
			RestEndpoint:     "https://sandbox-rest.ably.io",
		},
		Config: testAppConfig{
			Keys: []testAppKey{
				{},
			},
			Namespaces: []testAppNamespace{
				{ID: "persisted", Persisted: true},
			},
			Channels: []testAppChannels{
				{
					Name: "persisted:presence_fixtures",
					Presence: []testAppPresenceConfig{
						{ClientID: "client_bool", ClientData: "true"},
						{ClientID: "client_int", ClientData: "true"},
						{ClientID: "client_string", ClientData: "true"},
						{ClientID: "client_json", ClientData: "{ \"test\" => 'This is a JSONObject clientData payload'}"},
					},
				},
			},
		},
	}
}
