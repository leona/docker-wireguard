package main

import (
	"log"
	"os"
	"fmt"
)

var config *Config

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			SetLockFile("wireguard", false)
			os.Exit(1)
		}
	}()

	config = MakeConfig()
	wireman := MakeWireman("wg0", 51820)

	if config.DisableKillswitch {
		log.Println("WARNING: Kill switch disabled")
	}

	if config.MullvadAccount != "" {
		mullvad := MakeMullvad(config.MullvadAccount, MULLVAD_BLOCKING_MALWARE)
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
