//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017  Arduino AG (http://www.arduino.cc/)
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

	apt "github.com/arduino/go-apt-client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// AptListEvent sends a list of available packages and their status
func (s *Status) AptListEvent(client mqtt.Client, msg mqtt.Message) {
	// Get packages from system
	all, err := apt.List()
	if err != nil {
		s.Error("/apt/list/error", fmt.Errorf("Retrieving packages: %s", err))
		return
	}

	upgradable, err := apt.ListUpgradable()
	if err != nil {
		s.Error("/apt/list/error", fmt.Errorf("Retrieving packages: %s", err))
		return
	}

	// Prepare response payload
	type response struct {
		Packages   []*apt.Package `json:"packages"`
		Upgradable []*apt.Package `json:"upgradable"`
	}
	info := response{
		Packages:   all,
		Upgradable: upgradable,
	}

	// Send result
	data, err := json.Marshal(info)
	if err != nil {
		s.Error("/apt/list/error", fmt.Errorf("Json marshal result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/apt/list", string(data)+"\n")
}

// AptInstallEvent installs new packages
func (s *Status) AptInstallEvent(client mqtt.Client, msg mqtt.Message) {
	var params struct {
		Packages []string `json:"packages"`
	}
	err := json.Unmarshal(msg.Payload(), &params)
	if err != nil {
		s.Error("/apt/install/error", fmt.Errorf("Unmarshal '%s': %s", msg.Payload(), err))
		return
	}

	toInstall := []*apt.Package{}
	for _, p := range params.Packages {
		toInstall = append(toInstall, &apt.Package{Name: p})
	}
	out, err := apt.Install(toInstall...)
	s.InfoCommandOutput("/apt/install", out, err)
}

// AptUpdateEvent checks repositories for updates on installed packages
func (s *Status) AptUpdateEvent(client mqtt.Client, msg mqtt.Message) {
	out, err := apt.CheckForUpdates()
	s.InfoCommandOutput("/apt/update", out, err)
}

// AptUpgradeEvent installs upgrade for specified packages (or for all
// upgradable packages if none are specified)
func (s *Status) AptUpgradeEvent(client mqtt.Client, msg mqtt.Message) {
	var params struct {
		Packages []string `json:"packages"`
	}
	err := json.Unmarshal(msg.Payload(), &params)
	if err != nil {
		s.Error("/apt/upgrade/error", fmt.Errorf("Unmarshal '%s': %s", msg.Payload(), err))
		return
	}

	toUpgrade := []*apt.Package{}
	for _, p := range params.Packages {
		toUpgrade = append(toUpgrade, &apt.Package{Name: p})
	}

	if len(toUpgrade) == 0 {
		out, err := apt.UpgradeAll()
		s.InfoCommandOutput("/apt/upgrade", out, err)
	} else {
		out, err := apt.Upgrade(toUpgrade...)
		s.InfoCommandOutput("/apt/upgrade", out, err)
	}
}

// AptRemoveEvent deinstall the specified packages
func (s *Status) AptRemoveEvent(client mqtt.Client, msg mqtt.Message) {
	var params struct {
		Packages []string `json:"packages"`
	}
	err := json.Unmarshal(msg.Payload(), &params)
	if err != nil {
		s.Error("/apt/remove/error", fmt.Errorf("Unmarshal '%s': %s", msg.Payload(), err))
		return
	}

	toRemove := []*apt.Package{}
	for _, p := range params.Packages {
		toRemove = append(toRemove, &apt.Package{Name: p})
	}

	out, err := apt.Remove(toRemove...)
	s.InfoCommandOutput("/apt/remove", out, err)
}