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
* Logging with glog

We use glog library for logging in the project. In order to enable verbose logging the following needs to be done:

1. Make sure to import inittool in your go script, per this example:

<sup>
    import (
      . "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
    )
</sup>

2. Need to export the following SHELL variable:
> export ECO_VERBOSE_LEVEL=100

##### Notes:

  1. The value for the variable has to be >= 100.
  2. The variable can simply be exported in the shell where you run your automation.
  3. The go file you work on has to be in a directory under github.com/openshift-kni/eco-gotests/tests/ directory for being able to import inittools.
  4. Importing inittool also intializes the apiclient and it's available via "APIClient" variable.

* Collect logs from cluster with reporter

We use k8reporter library for collecting resource from cluster in case of test failure.
In order to enable k8reporter the following needs to be done:

1. Export DUMP_FAILED_TESTS and set it to true. Use example below
> export ECO_DUMP_FAILED_TESTS=true

2. Specify absolute path for logs directory like it appears below. By default /tmp/reports directory is used.
> export ECO_REPORTS_DUMP_DIR=/tmp/logs_directory

* Generation Polarion XML reports

We use polarion library for generating polarion compatible xml reports. 
The reporter is enabled by default and stores reports under REPORTS_DUMP_DIR directory.
In oder to disable polarion reporter the following needs to be done:
> export ECO_POLARION_REPORT=false


<!-- TODO Update this section with optional env vars for each test suite -->

## How to run

The test-runner [script](scripts/test-runner.sh) is the recommended way for executing tests.

Parameters for the script are controlled by the following environment variables:
- `ECO_TEST_FEATURES`: list of features to be tested ("all" will include all tests). All subdirectories under tests that match a feature will be included (internal directories are excluded) - _required_
- `ECO_TEST_LABELS`: ginkgo query passed to the label-filter option for including/excluding tests - _optional_ 
- `ECO_VERBOSE_SCRIPT`: prints verbose script information when executing the script - _optional_
- `ECO_TEST_VERBOSE`: executes ginkgo with verbose test output - _optional_
- `ECO_TEST_TRACE`: includes full stack trace from ginkgo tests when a failure occurs - _optional_

It is recommended to execute the runner script through the `make run-tests` make target.

Example:
```
$ export KUBECONFIG=/path/to/kubeconfig
$ export ECO_TEST_FEATURES="ztp kmm" 
$ export ECO_TEST_LABELS='platform-selection || image-service-statefulset'
$ make run-tests                    
Executing eco-gotests test-runner script
scripts/test-runner.sh
ginkgo -timeout=24h --keep-going --require-suite -r --label-filter="platform-selection || image-service-statefulset" ./tests/assisted/ztp ./tests/hw-accel/kmm
```
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

#### Functions format
If the function's arguments fit in a single line - use the following format:
```go
func Function(argInt1, argInt2 int, argString1, argString2 string) {
    ...
}
```

If the function's arguments don't fit in a single line - use the following format:
```go
func Function(
    argInt1 int,
    argInt2 int,
    argInt3 int,
    argInt4 int,
    argString1 string,
    argString2 string,
    argString3 string,
    argString4 string) {
    ...
}
```
One more acceptable format example:
```go
func Function(
    argInt1, argInt2 int, argString1, argString2 string, argSlice1, argSlice2 []string) {
	
}
```

### Common issues:
* If the automated commit check fails - make sure to pull/rebase the latest change and have a successful execution of 'make lint' locally first.
