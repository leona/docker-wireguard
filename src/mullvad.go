package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Mullvad struct {
	Account string
	BaseUrl string
	Servers *MullvadServers
	Keypair *WireguardKeypair
	Key     *MullvadKey
	DNS MullvadBlocking
}

type WireguardKeypair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type MullvadBlocking string

const (
	MULLVAD_BLOCKING_MALWARE MullvadBlocking = "100.64.0.4" // Malware blocking only
	MULLVAD_BLOCKING_AD_MALWARE MullvadBlocking = "100.64.0.5" // Ad and malware blocking, no tracker blocking
	MULLVAD_BLOCKING_TRACKER_MALWARE MullvadBlocking = "100.64.0.6" // Tracker and malware blocking, no ad blocking
	MULLVAD_BLOCKING_ALL MullvadBlocking = "100.64.0.7" // Ad, tracker and malware blocking (“everything”)
)

func MakeMullvad(account string, dns MullvadBlocking) *Mullvad {
	log.Println("Setting up Mullvad account:", account)
	mullvad := &Mullvad{
		Account: account,
		BaseUrl: "https://api.mullvad.net/",
		DNS: dns,
	}

	return mullvad
}

func (m *Mullvad) Get(endpoint string, response interface{}) error {
	request, err := http.NewRequest("GET", m.BaseUrl+endpoint, nil)

	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Token "+m.Account)

	client := &http.Client{}
	resp, err := client.Do(request)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Panic(err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Panic(err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Println("Failed to GET Mullvad API:", resp.StatusCode, "for", m.BaseUrl+endpoint)
		return errors.New(string(resp.StatusCode))
	}

	return nil
}

func (m *Mullvad) Post(endpoint string, data interface{}, response interface{}) error {
	body := &bytes.Buffer{}

	if err := json.NewEncoder(body).Encode(data); err != nil {
		log.Panic(err)
	}

	log.Println("sending", body.String())

	request, err := http.NewRequest("POST", m.BaseUrl+endpoint, body)

	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", "Token "+m.Account)

	client := &http.Client{}
	resp, err := client.Do(request)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	responseText := &bytes.Buffer{}
	responseText.ReadFrom(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Println("Failed POST to Mullvad API:", resp.Status, "body:", responseText.String())
		return errors.New(resp.Status)
	}

	if err := json.Unmarshal(responseText.Bytes(), &response); err != nil {
		return err
	}

	return nil
}

type MullvadServers struct {
	Countries []struct {
		Name   string `json:"name"`
		Code   string `json:"code"`
		Cities []struct {
			Name      string  `json:"name"`
			Code      string  `json:"code"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Relays    []struct {
				Hostname     string `json:"hostname"`
				Ipv4AddrIn   string `json:"ipv4_addr_in"`
				Ipv6AddrIn   string `json:"ipv6_addr_in"`
				PublicKey    string `json:"public_key"`
				MultihopPort int    `json:"multihop_port"`
			} `json:"relays"`
		} `json:"cities"`
	} `json:"countries"`
}

type MullvadKey struct {
	Ipv4Address string `json:"ipv4_address"`
	Ipv6Address string `json:"ipv6_address"`
	Id          string `json:"id"`
	Pubkey      string `json:"pubkey"`
}

func (m *Mullvad) GetServers() {
	log.Println("Getting Mullvad servers")
	var servers *MullvadServers

	if err := m.Get("public/relays/wireguard/v1/", &servers); err != nil {
		log.Panic("Failed to get Mullvad relays", err)
	}

	log.Println("Got Mullvad servers for", len(servers.Countries), "countries")
	m.Servers = servers
}

func (m *Mullvad) SetKeyPair() {
	log.Println("Checking Mullvad keypair")

	if _, err := os.Stat(config.ConfigPath + "/.mullvad.keypair"); errors.Is(err, os.ErrNotExist) {
		log.Println("Generating new Mullvad keypair")
		keypair, err := GenerateKeyPair()

		if err != nil {
			log.Panic(err)
		}

		file, err := os.Create(config.ConfigPath + "/.mullvad.keypair")

		if err != nil {
			log.Panic(err)
		}

		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(keypair); err != nil {
			log.Panic(err)
		}
	}

	file, err := os.Open(config.ConfigPath + "/.mullvad.keypair")

	if err != nil {
		log.Panic(err)
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	keypair := &WireguardKeypair{}

	if err := decoder.Decode(keypair); err != nil {
		log.Panic(err)
	}

	m.Keypair = keypair
}

type AddMullvadKey struct {
	Pubkey string `json:"pubkey"`
}

func (m *Mullvad) VerifyKeyPair() {
	var data *MullvadKey

	if err := m.Get("app/v1/wireguard-keys/"+m.Keypair.PublicKey, &data); err != nil || data.Id == "" {
		if err.Error() != string(http.StatusNotFound) {
			log.Panic("Failed to check keypair", err)
		}

		log.Println("Registering new Mullvad keypair", err)

		addKey := &AddMullvadKey{
			Pubkey: m.Keypair.PublicKey,
		}

		if err := m.Post("app/v1/wireguard-keys", addKey, &data); err != nil {
			log.Panic("Failed to register Mullvad keypair ", err)
		}
	}

	m.Key = data
	log.Println("Mullvad keypair validated:", m.Key)
}

func (m *Mullvad) CheckConfigs() {
	log.Println("Checking Mullvad configs")

	for _, country := range m.Servers.Countries {
		countryName := strings.ToLower(country.Name)

		if !stringInSlice(countryName, config.MullvadCountries) && !stringInSlice(country.Code, config.MullvadCountries) {
			continue
		}

		for _, city := range country.Cities {
			for _, relay := range city.Relays {
				configPath := config.ConfigPath + "/mullvad-" + relay.Hostname + ".conf"

				if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
					config := WireguardConfig{
						PrivateKey:      m.Keypair.PrivateKey,
						PublicKey:       relay.PublicKey,
						EndpointAddress: relay.Ipv4AddrIn,
						EndpointPort:    51820,
						AllowedIPs:      []string{"0.0.0.0/0"},
						Address:         m.Key.Ipv4Address,
						DNS: 					 string(m.DNS),
					}

					err := config.Save(configPath)
					FatalError(err)
					log.Println("Created Mullvad config for", relay.Hostname)
				}
			}
		}
	}
}

func GenerateKeyPair() (*WireguardKeypair, error) {
	key, err := wgtypes.GeneratePrivateKey()

	if err != nil {
		return nil, err
	}

	public := key.PublicKey()

	keypair := &WireguardKeypair{
		PrivateKey: key.String(),
		PublicKey:  public.String(),
	}

	return keypair, nil
}
