// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// Package cmd has the code for the following commands
//
//    Available Commands:
//      add         Add a proxy policy to an endpoint
//      clear       Remove all proxy policies from an endpoint
//      help        Help about any command
//      list        List the proxy policies on an endpoint
//      lookup      Report the ID of the HNS endpoint to which the specified container is attached
//      version     Output the version of hcnproxyctrl
//
package cmd

import (
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	proxy "github.com/microsoft/hcnproxyctrl/proxy"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "hcnproxyctrl.exe",
}

var (
	// VERSION is set during build
	VERSION string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show hcnproxyctrl version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(rootCmd.Use + " " + VERSION)
	},
}

// Flags for the "add" command
var (
	proxyPort   string
	userSID     string
	localAddr   string
	remoteAddr  string
	localPorts  string
	remotePorts string
	priority    uint16
	protocol    string
)

var cmdAdd = &cobra.Command{
	Use:   "add <HNS endpoint ID>",
	Short: "Add a proxy policy to an endpoint",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		endpointID := args[0]

		if userSID == "system" {
			userSID = proxy.LocalSystemSID
		}

		policy := proxy.Policy{
			ProxyPort:       proxyPort,
			UserSID:         userSID,
			LocalAddresses:  localAddr,
			RemoteAddresses: remoteAddr,
			LocalPorts:      localPorts,
			RemotePorts:     remotePorts,
			Priority:        priority,
		}

		err := proxy.AddPolicy(endpointID, policy)
		if err != nil {
			errorOut(err)
		}

		fmt.Println("Successfully added the policy")
	},
}

var cmdClear = &cobra.Command{
	Use:   "clear <HNS endpoint ID>",
	Short: "Remove all proxy policies from an endpoint",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		endpointID := args[0]
		numRemoved, err := proxy.ClearPolicies(endpointID)
		if err != nil {
			errorOut(err)
		}
		fmt.Println("Removed", numRemoved, "policies")
	},
}

var cmdList = &cobra.Command{
	Use:   "list <HNS endpoint ID>",
	Short: "List the proxy policies on an endpoint",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		endpointID := args[0]
		policies, err := proxy.ListPolicies(endpointID)
		if err != nil {
			errorOut(err)
		}
		spew.Dump(policies)
	},
}

// Flags for the "lookup" command
var (
	runtimeEndpoint string
)

var cmdLookup = &cobra.Command{
	Use:   "lookup <docker container ID>",
	Short: "Report the ID of the HNS endpoint to which the specified container is attached",
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		containerID := args[0]
		hnsEndpointID, err := proxy.GetEndpointFromContainer(containerID, runtimeEndpoint)
		if err != nil {
			errorOut(err)
		}
		fmt.Println(hnsEndpointID)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cmdAdd)
	rootCmd.AddCommand(cmdClear)
	rootCmd.AddCommand(cmdList)
	rootCmd.AddCommand(cmdLookup)

	// Flags for the "add" command
	cmdAdd.Flags().StringVar(&proxyPort, "port", "", "port the proxy is listening on")
	cmdAdd.MarkFlagRequired("port")
	cmdAdd.Flags().StringVar(&userSID, "usersid", "", `ignore traffic originating from the specified user SID (pass "system" to use the Local System SID)`)
	cmdAdd.Flags().StringVar(&localAddr, "localaddr", "", "only proxy traffic originating from the specified address")
	cmdAdd.Flags().StringVar(&remoteAddr, "remoteaddr", "", "only proxy traffic destinated to the specified address")
	cmdAdd.Flags().StringVar(&localPorts, "localports", "", "only proxy traffic originating from the specified port or port range")
	cmdAdd.Flags().StringVar(&remotePorts, "remoteports", "", "only proxy traffic destinated to the specified port or port range")
	cmdAdd.Flags().Uint16Var(&priority, "priority", 0, "the priority of this policy")

	// Flags for the "lookup" command
	cmdLookup.Flags().StringVar(&runtimeEndpoint, "runtimeendpoint", "", "CRI RuntimeEndpoint to query container information from")
}

func errorOut(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// Execute sets the version string, then calls through to Cobral Execute
func Execute(version string) {
	VERSION = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
