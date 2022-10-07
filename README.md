Ecosystem QE Golang Test Automation
=======
# Eco-gotests

## Overview
The [eco-gotests](https://github.com/openshift-kni/eco-gotests) is the downstream OCP telco/Ecosystem QE framework.
The project is based on golang+[ginkgo](https://onsi.github.io/ginkgo) framework.

### Project requirements
* golang v1.18.x
* ginkgo v2.x

## eco-gotests
The  [eco-gotests](https://github.com/openshift-kni/eco-gotests) is designed to test a pre-installed OCP cluster which meets the following requirements:

### Mandatory setup requirements:
* OCP cluster installed with version >=4.12

#### Optional:
* PTP operator
* SR-IOV operator
* SR-IOV-fec operator
* RAN DU profile

### Supported setups:
* Regular cluster 3 master nodes (VMs or BMs) 2 workers (VMs or BMs)
* Single Node Cluster (VM or BM)
* Public cloud (AWS)

**WARNING!**: [eco-gotests](https://github.com/openshift-kni/eco-gotests) removes existing configuration such as
PtpConfig, SR-IOV, SriovFecClusterConfig configs .

### General environment variables
#### Mandatory:
* `KUBECONFIG` - Path to kubeconfig file. Default: empty
#### Optional: 
<!-- TODO Update this section with optional env vars for each test suite -->

## How to run

# eco-gotests - How to contribute

The project uses a development method - forking workflow
### The following is a step-by-step example of forking workflow:
1) A developer [forks](https://docs.gitlab.com/ee/user/project/repository/forking_workflow.html#creating-a-fork)
   the [eco-gotests](https://github.com/openshift-kni/eco-gotests) project
2) A new local feature branch is created
3) A developer makes changes on the new branch.
4) New commits are created for the changes.
5) The branch gets pushed to developer's own server-side copy.
6) Changes are tested.
7) A developer opens a pull request(`PR`) from the new branch to
   the [eco-gotests](https://github.com/openshift-kni/eco-gotests).
8) The pull request gets approved from at least 2 reviewers for merge and is merged into
   the [eco-gotests](https://github.com/openshift-kni/eco-gotests) .

# eco-tests - Project structure
    .
    ├── config                             # config files
    ├── images                             # container images artifacts: Dockerfile ?
    ├── scripts                            # makefile scripts
    ├── tests                              # test cases directory
    │   ├── internal                       # common packages used acrossed framework
    │   │   ├── params                     # common constant and parameters used acrossed framework
    │   │   └── config                     # common config struct used across framework
    │   ├── cnf                            # cnf group test folder
    │   │   ├── network                    # networking test suites directory
    │   │   │   ├── cni                    # cni test suite directory 
    │   │   │   │   ├── internal           # internal packages used within cni test suite
    │   │   │   │   │    └── tsparams      # cni test suite constants and parameters package
    │   │   │   │   └── tests              # cni tests directory
    │   │   │   │        ├── common.go     # ginkgo common cni test functions
    │   │   │   │        ├── sysctl        # sysctl test cases
    │   │   │   │        │   ├──api.go     # api test cases
    │   │   │   │        │   ├──common.go  # common sysctl ginkngo function
    │   │   │   │        │   └──e2e.go     # e2e sysctl test cases
    │   │   │   │        └── vrf           # vrf test cases
    │   │   │   ├── dummy                  # dummy test suite directory 
    │   │   │   └── internal               # networking internal packages 
    │   │   │       └── netparam           # networking constants and parameters
    │   │   ├── internal                   # cnf internal packages 
    │   │   │   └── cnfparams              # cnf constants and parameters
    │   │   └── compute                    # compute test suites folder
    │   ├── external                       # external test cases from partners
    │   └── system                         # system group test folder
    ├── pkg                                # utils packages that later will be upstreamed
    │   ├── client
    │   ├── config
    │   ├── node
    │   ├── namespace
    │   └── pod
    └── vendors                            # Dependencies folder 
### Code conventions
#### Lint
Push requested are tested in a pipeline with golangci-lint. It is advised to add [Golangci-lint integration](https://golangci-lint.run/usage/integrations/) to your development editor. It's recommended to run `make lint` before uploading a PR.
