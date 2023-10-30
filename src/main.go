package main

import (
	"log"
)

var config *Config

func main() {
	config = MakeConfig()
	wireman := MakeWireman("wg0", 51820)

	if config.DisableKillswitch {
		log.Println("WARNING: Kill switch disabled")
	}

	if config.MullvadAccount != "" {
		mullvad := MakeMullvad(config.MullvadAccount)
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
