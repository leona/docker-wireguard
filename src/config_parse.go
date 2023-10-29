package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type WireguardConfig struct {
	PrivateKey      string
	PublicKey       string
	EndpointAddress string
	EndpointPort    int
	AllowedIPs      []string
	Address         string
}

func MakeWireguardConfigFromFile(configPath string) (*WireguardConfig, error) {
	log.Println("Reading wireguard config from:", configPath)
	file, err := os.Open(configPath)

	if err != nil {
		return nil, err
	}

	defer file.Close()
	config := &WireguardConfig{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "=")

		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		otherParts := strings.Join(parts[1:], "=")
		value := strings.TrimSpace(otherParts)

		switch key {
		case "PrivateKey":
			config.PrivateKey = value
		case "PublicKey":
			config.PublicKey = value
		case "AllowedIPs":
			config.AllowedIPs = []string{strings.Split(value, ",")[0]}
		case "Endpoint":
			split := strings.Split(value, ":")
			config.EndpointAddress = split[0]
			config.EndpointPort, _ = strconv.Atoi(split[1])
		case "Address":
			config.Address = strings.Split(value, ",")[0]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

func (w *WireguardConfig) Save(path string) error {
	log.Println("Saving wireguard config to:", path)
	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.WriteString(w.Serialize())

	if err != nil {
		return err
	}

	return nil
}

func (w *WireguardConfig) Serialize() string {
	return fmt.Sprintf(`[Interface]
PrivateKey=%s
Address=%s
[Peer]
PublicKey=%s
AllowedIPs=%s
Endpoint=%s`, w.PrivateKey, w.Address, w.PublicKey, strings.Join(w.AllowedIPs, ","), w.EndpointAddress+":"+strconv.Itoa(w.EndpointPort))
}
