package main

import (
	"os"
)

type Config struct {
	MullvadAccount    string
	MullvadCountries  []string
	ConfigPath        string
	DisableKillswitch bool
}

func MakeConfig() *Config {
	config := &Config{
		MullvadAccount:    DefaultString(os.Getenv("MULLVAD_ACCOUNT"), ""),
		MullvadCountries:  DefaultSlice(os.Getenv("MULLVAD_COUNTRIES"), []string{"nl"}),
		ConfigPath:        DefaultString(os.Getenv("CONFIG_PATH"), "/config"),
		DisableKillswitch: os.Getenv("DISABLE_KILLSWITCH") == "true",
	}

	return config
}
