package main

import (
	"log"
)

var config *Config

func main() {
	config = MakeConfig()
	wireman := MakeWireman("wg0", 51820)

	if config.MullvadAccount != "" {
		mullvad := MakeMullvad(config.MullvadAccount)

		if !config.DisableKillswitch {
			wireman.ToggleDNS(true)
			wireman.Allow(domainToIp(mullvad.BaseUrl))
		}

		mullvad.SetKeyPair()
		mullvad.VerifyKeyPair()
		mullvad.GetServers()
		mullvad.CheckConfigs()
	} else {
		log.Println("No Mullvad account provided, ignoring.")
	}

	configPath, err := GetRandomFile(config.ConfigPath, "conf")

	if err != nil || configPath == "" {
		log.Panic("Could not find any config files")
	}

	config, err := MakeWireguardConfigFromFile(configPath)
	FatalError(err)
	wireman.Up(config)
	wireman.TestTicker()
}
