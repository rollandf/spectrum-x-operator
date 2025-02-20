[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mellanox/spectrum-x-operator)](https://goreportcard.com/report/github.com/Mellanox/spectrum-x-operator)
[![Coverage Status](https://coveralls.io/repos/github/Mellanox/spectrum-x-operator/badge.svg)](https://coveralls.io/github/Mellanox/spectrum-x-operator)
[![Build, Test, Lint](https://github.com/Mellanox/spectrum-x-operator/actions/workflows/build-test-lint.yml/badge.svg?event=push)](https://github.com/Mellanox/spectrum-x-operator/actions/workflows/build-test-lint.yml)
[![CodeQL](https://github.com/Mellanox/spectrum-x-operator/actions/workflows/codeql.yml/badge.svg)](https://github.com/Mellanox/spectrum-x-operator/actions/workflows/codeql.yml)
[![Image push](https://github.com/Mellanox/spectrum-x-operator/actions/workflows/image-push-main.yml/badge.svg?event=push)](https://github.com/Mellanox/spectrum-x-operator/actions/workflows/image-push-main.yml)

# NVIDIA Spectrum-X Operator
Operator that Orchestrates NVIDIA Spectrum-X networking for Kubernetes.
This Operator is intended to be deployed via [NVIDIA Network Operator](https://github.com/Mellanox/network-operator)
and should not be regarded as a standalone component.

Network Operator Documentation can be found [here](https://docs.nvidia.com/networking/software/cloud-orchestration/index.html)

> __NOTE__: This project is currently under active development.


## Example:

```yaml
apiVersion: networking.example.com/v1
kind: SpectrumXNetwork
metadata:
  name: spectrum-x-network-config
spec:
  spectrumXEWNetworks:
    - name: rail_1_network
      railSubnet: "192.168.1.0/31"
      crossRailSubnet: "192.168.1.0/8"
    - name: rail_2_network
      railSubnet: "192.168.2.0/11"
      crossRailSubnet: "192.168.2.0/8"
  networkMappings:
    - rail: rail_1
      interface: eth0
    - rail: rail_2
      interface: eth1
  hosts:
    - hostID: host_1
      rails:
        - railName: rail_1
          railSubnet: "192.168.1.0/31"
          peerLeafPortIP: "192.168.1.1"
        - railName: rail_2
          railSubnet: "192.168.2.0/31"
          peerLeafPortIP: "192.168.2.1"
    - hostID: host_2
      rails:
        - railName: rail_1
          railSubnet: "192.168.1.0/31"
          peerLeafPortIP: "192.168.1.2"
        - railName: rail_2
          railSubnet: "192.168.2.0/31"
          peerLeafPortIP: "192.168.2.2"
```