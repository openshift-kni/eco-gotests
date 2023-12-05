# Ecosystem Edge Core Team - CNF Network

## Overview
Network tests are used for the testing of network operators and their features. Eco-gotests is a 
downstream CNF test framework utilizing a container test image with auxiliary resources for network testing. The CNF Telco 
Networking team is responsible for the maintenance and development of this repository. The test suite primary use is for
new GA and zStream releases regression testing.  

The Eco-gotest repo is in the process of replacing the [cnf-gotest](https://gitlab.cee.redhat.com/cnf/cnf-gotests) test cases. 
Upstream test cases can be found at [cnf-tests](https://github.com/openshift-kni/cnf-features-deploy/tree/master/cnf-tests). 

### Prerequisites for running these tests:

Eco-gotest suite is designed to test an OCP cluster version 4.13 and higher with the following pre-installed
CNF components:

* Machine config pool to define/collect configurations for labeled nodes
* SR-IOV operator
* MetalLB operator
* VRAN acceleration operator
* Performance Addon Operator
* SCTP via machine config

Container images used for network test cases:

* [eco-gotests-network-client](https://quay.io/repository/ocp-edge-qe/eco-gotests-network-client?tab=info)
* [eco-gotests-rootless-dpdk](https://quay.io/repository/ocp-edge-qe/eco-gotests-rootless-dpdk?tab=info)
* [eco-gotests-metallb-frr](https://quay.io/repository/ocp-edge-qe/frr?tab=info)

### Test suites:

| Name                                      | Description                                                       |
|-------------------------------------------|-------------------------------------------------------------------|
| [cni](cni/cni_suite_test.go)              | Tests that are run on corresponding operators                     |
| [day1day2](day1day2/day1day2_suite_test.go) | Tests are run on the sriov operator and existing sriov interfaces |
| [dpdk](dpdk/dpdk_suite_test.go)           | Tests are run on the sriov operator and existing sriov interfaces |
| [metallb](metallb/metallb_suite_test.go)  | Tests are run with metallb operator and frr container image       |
| [policy](policy/policy_suite_test.go)     | Tests that are run on corresponding operators                     |
| [sriov](sriov/sriov_suite_test.go)        | Tests are run with sriov operator and existing sriov interfaces   |

### Internal pkgs

| Name                                       | Description                                                       |
|--------------------------------------------|-------------------------------------------------------------------|
| [cmd](internal/cmd/cmd.go)               | Commands used to run on the test containers                    |
| [define](internal/define/nad.go) | Defines network attachment definitions for test containers |
| [netconfig](internal/netconfig/config.go)  | Configures environmental variables with default values |
| [netenv](internal/netenv/netenv.go)   | Verifies cluster configuration and support for sriov     |
| [netinittools](internal/netinittools/netinitools.go)    | Provides an APIClient for access to cluster                   |
| [netnmstate](internal/netnmstate/netnmstate.go)         | Commands to creates or recreates the new NMState instance and waits until its running   |
| [netparam](internal/netparam/const.go)         | Tests are run with sriov operator and existing sriov interfaces   |

### Eco-goinfra pkgs

The eco-goinfra project contains a collection of generic packages that can be used across various test projects. It was 
developed for collaboration between the different QE and Development teams. Utilizing the expertise of each team and to 
decrease the duplication of coding efforts.  Eco-infra project requires golang v1.19.x.

- [**README**](https://github.com/openshift-kni/eco-goinfra#readme)

### Inputs

Environment variables to change test image locations and worker labels:
- `ECO_CNF_CORE_NET_TEST_CONTAINER`: controls the location of the CNF test image.
- `ECO_CNF_CORE_NET_DPDK_TEST_CONTAINER`: controls the location of the DPDK test image.
- `ECO_CNF_CORE_NET_FRR_IMAGE`: controls the location of the FRR test image.
- `ECO_CNF_CORE_NET_CNF_MCP_LABEL`: variable used to identify the worker node label.

Please refer to the project README for a list of global inputs - [How to run](../../../README.md#how-to-run)
All network environmental variables can be found 'tests/cnf/core/network/internal/netconfig/config.go'

### Running Network Test Suites
```
# export KUBECONFIG=</path/to/hub/kubeconfig>
# export ECO_TEST_FEATURES=network
# export ECO_TEST_LABELS=net
# export ECO_TEST_VERBOSE='true'
# export ECO_VERBOSE_LEVEL=100
# export ECO_CNF_CORE_NET_VLAN=VLAN_ID
# export ECO_CNF_CORE_NET_SRIOV_INTERFACE_LIST=ens5f0,ens5f1
# export ECO_CNF_CORE_NET_MLB_ADDR_LIST=LIST of ip addresses # example 10.46.186.88,10.46.186.89,10.46.186.90,2620:52:0:2e51::88,2620:52:0:2e51::89,2620:52:0:2e51::90
# export ECO_CNF_CORE_NET_SWITCH_IP="switch_ip_address"
# export ECO_CNF_CORE_NET_SWITCH_INTERFACES=et-1/0/22,et-1/0/23,et-1/0/28,et-1/0/29
# export ECO_CNF_CORE_NET_SWITCH_USER="username"
# export ECO_CNF_CORE_NET_SWITCH_PASS='password'
# make run-tests
```

### Running Metallb Test Suites
```
# export KUBECONFIG=</path/to/hub/kubeconfig>
# export ECO_TEST_FEATURES=metallb
# export ECO_TEST_VERBOSE=true
# export ECO_VERBOSE_LEVEL=100
# export ECO_CNF_CORE_NET_MLB_ADDR_LIST=10.46.186.80,10.46.186.81,10.46.186.82
# make run-tests
```

### Additional Information

#### Lab Infrastructure:

The CNF Telco QE networking team has dedicated lab infrastructure for the execution of this test suite. The lab contains
multiple clusters with different types of SRIOV network cards and accelerators. The test cases are developed with this specific
network environment in mind. For generic network test cases the upstream [cnf-test](https://github.com/openshift-kni/cnf-features-deploy/tree/master/cnf-tests)
suite is used.

Two SRIOV interfaces from each card are connected to a managed lab switch. All non tagged packets are tagged with the native-vlan-id
and two trunk VLANs. VLANs may change between clusters.

* Switch configuration example
```
interfaces et-0/0/18
native-vlan-id 115;
mtu 9192;
unit 0 {
family ethernet-switching {
interface-mode trunk;
vlan {
members [ vlan115 vlan116 vlan117 ];
}
```


![pic](https://i.imgur.com/0jPXMdc.png)

* Hypervisor Master node
* Bare metal worker nodes
* SR-IOV NICs per worker nodes
* Managed switch connecting the SRIOV ports
* Jumbo frame support and configuration

