// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// Package cri implements calls to the CRI RuntimeEndpoint to get information about Containers.
package cri

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/util"
)

var (
	// RuntimeEndpoint is CRI server runtime endpoint
	RuntimeEndpoint string

	// Timeout  of connecting to server
	Timeout time.Duration
)

// CriParameters
type CriParameters struct {
	RuntimeEndpoint string
	Timeout         time.Duration
}

// DefaultCriParameters
func DefaultContainerdCriParameters() CriParameters {
	params := CriParameters{}
	params.RuntimeEndpoint = "tcp://127.0.0.1:2376"
	params.Timeout = 2 * time.Second
	return params
}

// ContainerInfo
type ContainerInfo struct {
	ContainerId string
	NamespaceId string
}

// ListContainers
func ListContainers(criParameters CriParameters) (containers []ContainerInfo, err error) {
	foundContainers := []ContainerInfo{}
	// Connect to the CRI Endpoint
	RuntimeEndpoint = criParameters.RuntimeEndpoint
	Timeout = criParameters.Timeout
	app := cli.NewApp()
	ctx := cli.NewContext(app, nil, nil)
	runtimeClient, runtimeConn, err := getRuntimeClient(ctx)
	if err != nil {
		return nil, err
	}
	defer closeConnection(ctx, runtimeConn)

	request := &pb.ListContainersRequest{}
	response, err := runtimeClient.ListContainers(context.Background(), request)
	if err != nil {
		return nil, err
	}

	criContainers := response.GetContainers()
	for _, container := range criContainers {
		containerStatusRequest := &pb.ContainerStatusRequest{
			ContainerId: container.Id,
			Verbose:     true, // Populates the info json
		}
		containerStatusResponse, err := runtimeClient.ContainerStatus(context.Background(), containerStatusRequest)
		if err != nil {
			return nil, err
		}

		// Read the info json
		info := containerStatusResponse.Info["info"]
		var infoMap map[string]interface{}
		json.Unmarshal([]byte(info), &infoMap)

		runtimeSpec := infoMap["runtimeSpec"].(map[string]interface{})
		windows := runtimeSpec["windows"].(map[string]interface{})
		network := windows["network"].(map[string]interface{})
		networkNamespace := network["networkNamespace"].(string)

		foundContainer := ContainerInfo{
			ContainerId: container.Id,
			NamespaceId: networkNamespace,
		}
		foundContainers = append(foundContainers, foundContainer)
	}

	return foundContainers, nil
}

// Copied from https://github.com/kubernetes-sigs/cri-tools/cmd/crictl/util.go

func getRuntimeClient(context *cli.Context) (pb.RuntimeServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getRuntimeClientConnection(context)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %v", err)
	}

	runtimeClient := pb.NewRuntimeServiceClient(conn)
	return runtimeClient, conn, nil
}

func closeConnection(context *cli.Context, conn *grpc.ClientConn) error {
	if conn == nil {
		return nil
	}
	return conn.Close()
}

// Copied from https://github.com/kubernetes-sigs/cri-tools/cmd/crictl/main.go

func getRuntimeClientConnection(context *cli.Context) (*grpc.ClientConn, error) {
	addr, dialer, err := util.GetAddressAndDialer(RuntimeEndpoint)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(Timeout), grpc.WithDialer(dialer))
	if err != nil {
		return nil, fmt.Errorf("failed to connect, make sure you are running as root and the runtime has been started: %v", err)
	}
	return conn, nil
}
