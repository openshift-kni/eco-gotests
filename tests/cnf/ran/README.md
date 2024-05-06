# Ecosystem Telco FT Team - CNF vRAN

## Overview

CNF vRAN tests clusters for use in vRAN deployments.

### Prerequisites for running these tests

Designed for supported OCP versions with the following installed:

* Core OCP
* DU profile

Some test suites (ZTP, TALM) require clusters to be deployed via GitOps ZTP and managed by a hub cluster.

### Test suites

| Name                                                             | Description                                          |
|------------------------------------------------------------------|------------------------------------------------------|
| [containernshide](containernshide/containernshide_suite_test.go) | Tests that containers have a hidden mount namespace. |

### Internal pkgs

| Name                                                 | Description                                                       |
|------------------------------------------------------|-------------------------------------------------------------------|
| [ranconfig](internal/ranconfig/config.go)            | Configures environment variables and default values               |
| [raninittools](internal/raninittools/raninitools.go) | Provides an APIClient for access to cluster                       |
| [ranparam](internal/ranparam/const.go)               | Labels and other constants used in the test suites                |

### Eco-goinfra pkgs

[**README**](https://github.com/openshift-kni/eco-goinfra#readme)

### Inputs

Currently there are no specific inputs for RAN tests.

Please refer to the project README for a list of global inputs - [How to run](../../../README.md#how-to-run)

### Running the RAN test suites

```
# export KUBECONFIG=</path/to/spoke/kubeconfig>
# export ECO_TEST_FEATURES=ran
# export ECO_TEST_VERBOSE=true
# export ECO_VERBOSE_LEVEL=100
# make run-tests
```

### Additional Information
