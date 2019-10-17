// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// Package main builds an executable that allows users to program layer-4 proxy
// policies on Windows through the Host Networking Service (HNS).
//
//  Usage:
//    hcnproxyctrl.exe [command]
//
//    Available Commands:
//      add         Add a proxy policy to an endpoint
//      clear       Remove all proxy policies from an endpoint
//      help        Help about any command
//      list        List the proxy policies on an endpoint
//      lookup      Report the ID of the HNS endpoint to which the specified container is attached
//      version     Output the version of hcnproxyctrl
//
//    Flags:
//      -h, --help   help for hcnproxyctrl.exe
//
//    Use "hcnproxyctrl.exe [command] --help" for more information about a command.
package main

import (
	"github.com/Microsoft/hcnproxyctrl/cmd"
)

var (
	// VERSION is set during build
	VERSION = "0.0.1"
)

func main() {
	cmd.Execute(VERSION)
}
