// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

// Package hcnproxyctrl implements a high-level library that allows users to program
// layer-4 proxy policies on Windows through the Host Networking Service (HNS).
package hcnproxyctrl

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/Microsoft/hcsshim/hcn"
	cri "github.com/microsoft/hcnproxyctrl/cri"
)

// LocalSystemSID defines the SID of the permission set known in Windows
// as "Local System". In a sidecar proxy deployment, users will typically run
// the proxy container under that SID, and assign it to the UserSID field of
// the Policy struct, to signify to HNS that traffic originating from that SID
// should not be forwarded to the proxy -- which would create a loop, since
// traffic originating from the proxy would be forwarded back to the proxy.
const LocalSystemSID = "S-1-5-18"

// Policy specifies the proxy and the kind of traffic that will be
// intercepted by the proxy.
type Policy struct {
	// The port the proxy is listening on. (Required)
	ProxyPort string

	// Ignore traffic originating from the specified user SID. (Optional)
	UserSID string

	// Only proxy traffic originating from the specified address. (Optional)
	LocalAddresses string

	// Only proxy traffic destinated to the specified address. (Optional)
	RemoteAddresses string

	// Only proxy traffic originating from the specified port or port range. (Optional)
	LocalPorts string

	// Only proxy traffic destinated to the specified port or port range. (Optional)
	RemotePorts string

	// The priority of this policy. (Optional)
	// For more info, see https://docs.microsoft.com/en-us/windows/win32/fwp/filter-weight-assignment.
	Priority uint16

	// Only proxy traffic using this protocol. TCP is the only supported
	// protocol for now, and this field defaults to that if left blank. (Optional)
	// Ex: 6 = TCP
	Protocol string
}

// AddPolicy adds a layer-4 proxy policy to HNS. The endpointID refers to the
// ID of the endpoint as defined by HNS (eg. the GUID output by hnsdiag).
// An error is returned if the policy passed in argument is invalid, or if it
// could not be applied for any reason.
func AddPolicy(hnsEndpointID string, policy Policy) error {
	if err := validatePolicy(policy); err != nil {
		return err
	}

	// TCP is the default protocol and is the only supported one anyway.
	policy.Protocol = "6"

	policySetting := hcn.L4WfpProxyPolicySetting{
		Port:    policy.ProxyPort,
		UserSID: policy.UserSID,
		FilterTuple: hcn.FiveTuple{
			LocalAddresses:  policy.LocalAddresses,
			RemoteAddresses: policy.RemoteAddresses,
			LocalPorts:      policy.LocalPorts,
			RemotePorts:     policy.RemotePorts,
			Protocols:       policy.Protocol,
			Priority:        policy.Priority,
		},
	}

	policyJSON, err := json.Marshal(policySetting)
	if err != nil {
		return err
	}

	endpointPolicy := hcn.EndpointPolicy{
		Type:     hcn.L4WFPPROXY,
		Settings: policyJSON,
	}

	request := hcn.PolicyEndpointRequest{
		Policies: []hcn.EndpointPolicy{endpointPolicy},
	}

	endpoint, err := hcn.GetEndpointByID(hnsEndpointID)
	if err != nil {
		return err
	}

	return endpoint.ApplyPolicy(hcn.RequestTypeAdd, request)
}

// ListPolicies returns the proxy policies that are currently active on the
// given endpoint.
func ListPolicies(hnsEndpointID string) ([]Policy, error) {
	hcnPolicies, err := listPolicies(hnsEndpointID)
	if err != nil {
		return nil, err
	}

	var policies []Policy
	for _, hcnPolicy := range hcnPolicies {
		policies = append(policies, hcnPolicyToAPIPolicy(hcnPolicy))
	}

	return policies, nil
}

// ClearPolicies removes all the proxy policies from the specified endpoint.
// It returns the number of policies that were removed, which will be zero
// if an error occurred or if the endpoint did not have any active proxy policies.
func ClearPolicies(hnsEndpointID string) (numRemoved int, err error) {
	policies, err := listPolicies(hnsEndpointID)
	if err != nil {
		return 0, err
	}

	policyReq := hcn.PolicyEndpointRequest{
		Policies: policies,
	}

	policyJSON, err := json.Marshal(policyReq)
	if err != nil {
		return 0, err
	}

	modifyReq := &hcn.ModifyEndpointSettingRequest{
		ResourceType: hcn.EndpointResourceTypePolicy,
		RequestType:  hcn.RequestTypeRemove,
		Settings:     policyJSON,
	}

	return len(policies), hcn.ModifyEndpointSettings(hnsEndpointID, modifyReq)
}

// GetEndpointFromContainer takes a container ID as argument and returns
// the ID of the HNS endpoint to which it is attached. It returns an error if
// the specified container is not attached to any endpoint.
// Note: there is no verification that the ID passed as argument belongs
// to an actual container.
func GetEndpointFromContainer(containerID string, runtimeEndpoint string) (hnsEndpointID string, err error) {
	params := cri.DefaultContainerdCriParameters()
	if len(runtimeEndpoint) > 0 {
		params.RuntimeEndpoint = runtimeEndpoint
	}
	containers, err := cri.ListContainers(params)
	if err != nil {
		return "", err
	}
	var namespaceID string
	for _, container := range containers {
		if container.ContainerId == containerID {
			namespaceID = container.NamespaceId
		}
	}
	if len(namespaceID) == 0 {
		return "", errors.New("could not find the container")
	}

	endpointIDs, err := hcn.GetNamespaceEndpointIds(namespaceID)
	if err != nil {
		return "", err
	}
	if len(endpointIDs) == 0 {
		return "", errors.New("could not find an endpoint attached to that container")
	}

	return strings.Join(endpointIDs, ","), nil
}

// listPolicies returns the HCN *proxy* policies that are currently active on the
// given endpoint.
func listPolicies(hnsEndpointID string) ([]hcn.EndpointPolicy, error) {
	endpoint, err := hcn.GetEndpointByID(hnsEndpointID)
	if err != nil {
		return nil, err
	}

	var policies []hcn.EndpointPolicy
	for _, policy := range endpoint.Policies {
		if policy.Type == hcn.L4WFPPROXY {
			policies = append(policies, policy)
		}
	}

	return policies, nil
}

// hcnPolicyToAPIPolicy converts an L4 proxy policy as defined by hcsshim
// to our own API.
func hcnPolicyToAPIPolicy(hcnPolicy hcn.EndpointPolicy) Policy {
	if hcnPolicy.Type != hcn.L4WFPPROXY {
		panic("not an L4 proxy policy")
	}

	// Assuming HNS will never return invalid values from here.
	var hcnPolicySetting hcn.L4WfpProxyPolicySetting
	_ = json.Unmarshal(hcnPolicy.Settings, &hcnPolicySetting)

	return Policy{
		ProxyPort:       hcnPolicySetting.Port,
		UserSID:         hcnPolicySetting.UserSID,
		LocalAddresses:  hcnPolicySetting.FilterTuple.LocalAddresses,
		RemoteAddresses: hcnPolicySetting.FilterTuple.RemoteAddresses,
		LocalPorts:      hcnPolicySetting.FilterTuple.LocalPorts,
		RemotePorts:     hcnPolicySetting.FilterTuple.RemotePorts,
		Priority:        hcnPolicySetting.FilterTuple.Priority,
		Protocol:        hcnPolicySetting.FilterTuple.Protocols,
	}
}

// validatePolicy returns nil iff the provided policy is valid.
// For now it only checks that the port number is nonzero.
func validatePolicy(policy Policy) error {
	if len(policy.ProxyPort) == 0 {
		return errors.New("policy missing proxy port")
	}
	port, _ := strconv.Atoi(policy.ProxyPort)
	if port == 0 {
		return errors.New("policy has invalid proxy port value: 0")
	}
	return nil
}

// formatIP returns the given address as a string,
// or the empty string if it's nil.
func formatIP(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}
