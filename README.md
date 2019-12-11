# hcnproxyctl

Host Container Networking Proxy Controller is a high-level library and executable that allows
users to program layer-4 proxy policies on Windows through the Host Networking Service (HNS).

It is intended to be used as part of the Service Mesh Interface to program the proxy of traffic into the sidecar.
* [SMI Release Announcement](https://cloudblogs.microsoft.com/opensource/2019/05/21/service-mesh-interface-smi-release/) 
* [SMI Spec](https://smi-spec.io/)

## Example - Golang

The following go code sets a proxy policy on the endpoint attached to a known Docker
container, such that:

- Outbound TCP traffic will be redirected through port 8000
- Unless it originates from the proxy itself, which is running as the specified User SID

```go
containerID := "ccaae3aba155ccfb08fdf2e51fa4034f19db40d0ae5b485820544e49a60499c0"
hnsEndpointID, _ := hcnproxyctl.GetEndpointFromContainer(containerID, nil)

proxyPolicy := hcnproxyctl.Policy{
        Port: 8000,
        UserSID: "S-1-5-21-1688553208-1784504425-564974220-1000",
}

_ = hcnproxyctl.AddPolicy(hnsEndpointID, proxyPolicy)
```

## Example - hcnproxyctl.exe

The following code sets a proxy policy on the endpoint attached to a known Docker 
container, such that:

- Outbound TCP traffic will be redirected through port 8000
- Unless it originates from the proxy itself, which is running the specified User SID

```powershell
> .\crictl.exe ps
CONTAINER ID        IMAGE                                       CREATED             STATE               NAME                 ATTEMPT             POD ID
ccaae3aba155c       mcr.microsoft.com/windows/nanoserver:1809   24 hours ago        Running             windows-hello-test   0                   f04bda79168c8

> .\crictl.exe inspect ccaae3aba155c
{
  "status": {
    "id": "ccaae3aba155ccfb08fdf2e51fa4034f19db40d0ae5b485820544e49a60499c0",
...

> .\hcnproxyctl.exe lookup ccaae3aba155ccfb08fdf2e51fa4034f19db40d0ae5b485820544e49a60499c0
93f86a7f-e361-4362-b8a4-81bbb6a622dd

> .\hcnproxyctl.exe add 93f86a7f-e361-4362-b8a4-81bbb6a622dd --port 8000 --usersid S-1-5-21-1688553208-1784504425-564974220-1000
Successfully added the policy

> .\hcnproxyctl.exe list 93f86a7f-e361-4362-b8a4-81bbb6a622dd
([]proxyctl.Policy) (len=1 cap=1) {
 (proxyctl.Policy) {
  ProxyPort: (uint16) 8000,
  UserSID: (string) (len=45) "S-1-5-21-1688553208-1784504425-564974220-1000",
  LocalAddr: (net.IP) <nil>,
  RemoteAddr: (net.IP) <nil>,
  Priority: (uint8) 0,
  Protocol: (proxyctl.Protocol) 6
 }
}
```

## Current limitations

As of October, 2019, these are the limitations of hcnproxyctl (subject to change):

- Outbound traffic only.
- TCP traffic only.
- IPv4 traffic only.
- No multi-proxy support. Only one proxy application per pod.

## Dependencies

This project relies on the [hcsshim](https://github.com/microsoft/hcsshim).

This project requires an HNS feature availible on Windows insider builds.

The "Lookup" command assumes that containers were created by a CRI compatable container manager. Ex: ContainerD

For system requirements to run this project, see the Microsoft docs on [Windows Container requirements](https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/system-requirements).

## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
