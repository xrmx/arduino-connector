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

	apt "github.com/arduino/go-apt-client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	docker "github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func checkAndInstallDocker() {
	// fmt.Println("try to install docker-ce")
	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println("Docker daemon not found!")
		fmt.Println(err.Error())
		// err should be "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"
	}
	_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		fmt.Println("Docker daemon not found!")
		fmt.Println(err.Error())
	}

	if err != nil {
		go func() {
			// dpkg --configure -a for prevent block of installation
			dpkgCmd := exec.Command("dpkg", "--configure", "-a")
			if out, err := dpkgCmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to reconfigure dpkg:")
				fmt.Println(string(out))
			}

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
				fmt.Println("Failed to set up the stable docker repository:")
				fmt.Println(string(out))
			}

			apt.CheckForUpdates()
			toInstall := &apt.Package{Name: "docker-ce"}
			if out, err := apt.Install(toInstall); err != nil {
				fmt.Println("Failed to install docker-ce:")
				fmt.Println(string(out))
				return
			}

			// systemctl enable docker
			sysCmd := exec.Command("systemctl", "enable", "docker")
			if out, err := sysCmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to systemctl enable docker:")
				fmt.Println(string(out))
			}

			// fmt.Println("done to install docker-ce")
		}()
	}

}

// ContainersPsEvent implements docker ps
func (s *Status) ContainersPsEvent(client mqtt.Client, msg mqtt.Message) {
	cli, err := docker.NewEnvClient()
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marshal result: %s", err))
		return
	}

	// Send result
	data, err := json.Marshal(containers)
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/containers/ps", string(data)+"\n")
}

// ContainersListImagesEvent implements docker images
func (s *Status) ContainersListImagesEvent(client mqtt.Client, msg mqtt.Message) {
	// Gather images
	cli, err := docker.NewEnvClient()
	if err != nil {
		s.Error("/containers/images", fmt.Errorf("images result: %s", err))
		return
	}

	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		s.Error("/containers/images", fmt.Errorf("images result: %s", err))
		return
	}

	// Send result
	data, err := json.Marshal(images)
	if err != nil {
		s.Error("/containers/images", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/containers/images", string(data)+"\n")
}

// ContainersRunEvent implements docker images
func (s *Status) ContainersActionEvent(client mqtt.Client, msg mqtt.Message) {
	var runParams struct {
		ImageName     string `json:"image"`
		ContainerName string `json:"name"`
		ContainerId   string `json:"id"`
		RunAsDaemon   bool   `json:"background"`
		Action        string `json:"action"`
	}

	type RunPayload struct {
		ImageName     string `json:"image"`
		ContainerName string `json:"name"`
		RunAsDaemon   bool   `json:"background"`
		ContainerId   string `json:"id"`
		Action        string `json:"action"`
	}

	runResponse := RunPayload{}

	err := json.Unmarshal(msg.Payload(), &runParams)
	if err != nil {
		s.Error("/containers/action", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}

	ctx := context.Background()
	cli, err := docker.NewEnvClient()
	if err != nil {
		s.Error("/containers/action", fmt.Errorf("run result: %s", err))
		return
	}

	if runParams.Action == "start" {
		// out, err := cli.ImagePull(ctx, runParams.ImageName, types.ImagePullOptions{})
		_, err := cli.ImagePull(ctx, runParams.ImageName, types.ImagePullOptions{})
		if err != nil {
			s.Error("/containers/action", fmt.Errorf("image pull result: %s", err))
			return
		}

		//io.Copy(os.Stdout, out)

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Image: runParams.ImageName,
		}, nil, nil, runParams.ContainerName)
		if err != nil {
			s.Error("/containers/action", fmt.Errorf("container create result: %s", err))
			return
		}

		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			s.Error("/containers/action", fmt.Errorf("container start result: %s", err))
			return
		}
		runResponse = RunPayload{
			ImageName:     runParams.ImageName,
			ContainerName: runParams.ContainerName,
			RunAsDaemon:   runParams.RunAsDaemon,
			ContainerId:   resp.ID,
			Action:        runParams.Action,
		}

	} else if runParams.Action == "stop" {

		if err := cli.ContainerStop(ctx, runParams.ContainerId, nil); err != nil {
			s.Error("/containers/action", fmt.Errorf("container action result: %s", err))
			return
		}

		runResponse = RunPayload{
			ImageName:     runParams.ImageName,
			ContainerName: runParams.ContainerName,
			RunAsDaemon:   runParams.RunAsDaemon,
			ContainerId:   "",
			Action:        runParams.Action,
		}
	}

	// Send result
	data, err := json.Marshal(runResponse)
	if err != nil {
		s.Error("/containers/action", fmt.Errorf("Json marshal result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/containers/action", string(data)+"\n")

}