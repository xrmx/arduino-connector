//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2018  Arduino AG (http://www.arduino.cc/)
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	apt "github.com/arduino/go-apt-client"
	"github.com/arduino/go-system-stats/disk"
	"github.com/arduino/go-system-stats/mem"
	"github.com/arduino/go-system-stats/network"
	"github.com/docker/docker/api/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	docker "github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// WiFiEvent tries to connect to the specified wifi network
func (s *Status) WiFiEvent(client mqtt.Client, msg mqtt.Message) {
	// try registering a new wifi network
	var info struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}
	err := json.Unmarshal(msg.Payload(), &info)
	if err != nil {
		s.Error("/wifi", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}
	net.AddWirelessConnection(info.SSID, info.Password)
}

// EthEvent tries to change IP/Netmask/DNS configuration of the wired connection
func (s *Status) EthEvent(client mqtt.Client, msg mqtt.Message) {
	// try registering a new wifi network
	var info net.IPProxyConfig
	err := json.Unmarshal(msg.Payload(), &info)
	if err != nil {
		s.Error("/ethernet", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}
	net.AddWiredConnection(info)
}

func checkAndInstallNetworkManager() {
	_, err := net.GetNetworkStats()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "NetworkManager") {
		go func() {
			toInstall := &apt.Package{Name: "network-manager"}
			if out, err := apt.Install(toInstall); err != nil {
				fmt.Println("Failed to install network-manager:")
				fmt.Println(string(out))
				return
			}
			cmd := exec.Command("/etc/init.d/network-manager", "start")
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to start network-manager:")
				fmt.Println(string(out))
			}
		}()
	}
}

func checkAndInstallDocker() {
	// fmt.Println("try to install docker-ce")
	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println("Docker daemon not found!")
		fmt.Println(err.Error())
	}
	_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		fmt.Println("Docker daemon not found!")
		fmt.Println(err.Error())
	}

	if err != nil {
		go func() {
			//steps from https://docs.docker.com/install/linux/docker-ce/ubuntu/
			apt.CheckForUpdates()
			dockerPrerequisitesPackages := []*apt.Package{&apt.Package{Name: "apt-transport-https"}, &apt.Package{Name: "ca-certificates"}, &apt.Package{Name: "curl"}, &apt.Package{Name: "software-properties-common"}}
			for _, pac := range dockerPrerequisitesPackages {
				if out, err := apt.Install(pac); err != nil {
					fmt.Println("Failed to install: ", pac.Name)
					fmt.Println(string(out))
					return
				}
			}
			cmdString := "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -"
			cmd := exec.Command("bash", "-c", cmdString)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to add Docker’s official GPG key:")
				fmt.Println(string(out))
			}

			repoString := "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
			cmd = exec.Command("add-apt-repository", repoString)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to set up the stable repository:")
				fmt.Println(string(out))
			}

			apt.CheckForUpdates()
			toInstall := &apt.Package{Name: "docker-ce"}
			if out, err := apt.Install(toInstall); err != nil {
				fmt.Println("Failed to install docker-ce:")
				fmt.Println(string(out))
				return
			}
			// fmt.Println("done to install docker-ce")
		}()
	}

}

// StatsEvent sends statistics about resource used in the system (RAM, Disk, Network, etc...)
func (s *Status) StatsEvent(client mqtt.Client, msg mqtt.Message) {
	// Gather all system data metrics
	memStats, err := mem.GetStats()
	if err != nil {
		s.Error("/stats", fmt.Errorf("Retrieving memory stats: %s", err))
	}

	diskStats, err := disk.GetStats()
	if err != nil {
		s.Error("/stats", fmt.Errorf("Retrieving disk stats: %s", err))
	}

	netStats, err := net.GetNetworkStats()
	if err != nil {
		s.Error("/stats", fmt.Errorf("Retrieving network stats: %s", err))
	}

	type StatsPayload struct {
		Memory  *mem.Stats      `json:"memory"`
		Disk    []*disk.FSStats `json:"disk"`
		Network *net.Stats      `json:"network"`
	}

	info := StatsPayload{
		Memory:  memStats,
		Disk:    diskStats,
		Network: netStats,
	}

	// Send result
	data, err := json.Marshal(info)
	if err != nil {
		s.Error("/stats", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/stats", string(data)+"\n")
}
