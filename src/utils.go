package main

import (
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func SetLockFile(name string, state bool) {
	path := "/tmp/" + name + ".lock"

	if state {
		log.Println("Creating lock file:", name)
		os.Create(path)
	} else {
		log.Println("Removing lock file:", name)
		os.Remove(path)
	}
}

func DefaultString(input string, defaultValue string) string {
	if input == "" {
		return defaultValue
	}
	return input
}

func FatalError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func DefaultInt(input string, defaultValue int) int {
	if input == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(input)

	if err != nil {
		return defaultValue
	}

	return i
}

func DefaultSlice(input string, defaultValue []string) []string {
	if input == "" {
		return defaultValue
	}

	split := strings.Split(input, ",")

	for i, item := range split {
		split[i] = strings.ToLower(strings.TrimSpace(item))
	}

	return split
}

func stringInSlice(str string, list []string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}

	return false
}

func roundFloat64(input float64, places int) float64 {
	rounding := 1.0

	for i := 0; i < places; i++ {
		rounding *= 10.0
	}

	return float64(int(input*rounding)) / rounding
}

func GetRandomFile(path string, extension string) (string, error) {
	var files []string

	// read the files in the directory
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() { // skip if it is a directory
			if filepath.Ext(path) == "."+extension {
				files = append(files, path)
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	// select a random file
	rand.Seed(time.Now().Unix())
	randomFile := files[rand.Intn(len(files))]

	return randomFile, nil
}

func domainToIp(value string) string {
	log.Println("Getting IP for:", value)
	url, err := url.Parse(value)
	FatalError(err)
	hostname := url.Hostname()
	log.Println("Got hostname:", hostname)
	ips, err := net.LookupIP(hostname)
	FatalError(err)

	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ip := ipv4.String()
			log.Println("Got ip:", ip)
			return ip
		}
	}

	log.Println("failed to get IP")
	return ""
}
