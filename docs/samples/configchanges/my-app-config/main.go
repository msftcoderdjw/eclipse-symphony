package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

const configFilePath = "/etc/config/myConfigFile"

func readConfigFile() (string, error) {
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	fmt.Println("Starting my-app-config v2.0 ...")
	for {
		content, err := readConfigFile()
		fmt.Println("Current time:", time.Now())
		if err != nil {
			log.Printf("Error reading config file: %v", err)
		} else {
			fmt.Println("Config file content:")
			fmt.Println(content)
		}

		fmt.Println("Environment variables beginning with MY_APP_:")
		// print all environemnt variables beginning with MY_APP_
		for _, e := range os.Environ() {
			pair := strings.Split(e, "=")
			if strings.HasPrefix(pair[0], "MY_APP_") {
				fmt.Println(strings.Join(pair, "="))
			}
		}

		fmt.Println("Waiting for 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}
