module github.com/openshift-kni/eco-gotests

go 1.22

require (
	github.com/Juniper/go-netconf v0.3.0
	github.com/NVIDIA/gpu-operator v1.11.1
	github.com/cavaliergopher/cpio v1.0.1
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/golang/glog v1.2.1
	github.com/hashicorp/go-version v1.7.0
	github.com/k8snetworkplumbingwg/multi-networkpolicy v0.0.0-20240528155521-f76867e779b8
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.7.0
	github.com/k8snetworkplumbingwg/sriov-network-operator v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/metal3-io/baremetal-operator/apis v0.6.1
	github.com/nmstate/kubernetes-nmstate/api v0.0.0-20240605150941-df565dd7bf35
	github.com/onsi/ginkgo/v2 v2.19.0
	github.com/onsi/gomega v1.33.1
	github.com/openshift-kni/eco-goinfra v0.0.0-20240626143745-4d019aeb6f02 // release-4.15
	github.com/openshift-kni/k8sreporter v1.0.5
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/assisted-service/api v0.0.0
	github.com/openshift/assisted-service/models v0.0.0
	github.com/openshift/cluster-nfd-operator v0.0.0-20240604082319-19bf50784aa7
	github.com/openshift/cluster-node-tuning-operator v0.0.0-20240606084543-6d2e11aec345
	github.com/openshift/hive/apis v0.0.0-20220707210052-4804c09ccc5a
	github.com/openshift/machine-config-operator v0.0.1-0.20230811181556-63d7be1ef18b
	github.com/operator-framework/api v0.23.0
	github.com/rh-ecosystem-edge/kernel-module-management v0.0.0-20240605101434-e1de2798b3c4
	github.com/walle/targz v0.0.0-20140417120357-57fe4206da5a
	github.com/wk8/go-ordered-map/v2 v2.1.8
	golang.org/x/exp v0.0.0-20240604190554-fc45aab8b7f8
	gopkg.in/k8snetworkplumbingwg/multus-cni.v4 v4.0.2
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.29.5
	k8s.io/apimachinery v0.29.5
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.17.5
)

require (
	github.com/kedacore/keda-olm-operator v0.0.0-20240501182040-762f6be5a942
	github.com/kedacore/keda/v2 v2.14.0
	github.com/openshift-kni/lifecycle-agent v0.0.0-20240606123201-0c45cd13c2f1
	github.com/openshift-kni/numaresources-operator v0.4.16-0rc0
	github.com/openshift/client-go v0.0.1
	github.com/openshift/installer v0.0.0-00010101000000-000000000000
	github.com/povsister/scp v0.0.0-20210427074412-33febfd9f13e
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.73.2
	github.com/stretchr/testify v1.9.0
	github.com/vmware-tanzu/velero v1.13.2
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57
	open-cluster-management.io/config-policy-controller v0.13.0
	open-cluster-management.io/governance-policy-propagator v0.13.0
	open-cluster-management.io/multicloud-operators-subscription v0.13.0
)

require (
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/expr-lang/expr v1.16.5 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/kdomanski/iso9660 v0.2.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/metal3-io/baremetal-operator/pkg/hardwareutils v0.5.1 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/nutanix-cloud-native/prism-go-client v0.2.1-0.20220804130801-c8a253627c64 // indirect
	github.com/openshift/cluster-logging-operator v0.0.0-20240606085930-750f369019d4 // indirect
	github.com/openshift/elasticsearch-operator v0.0.0-20220613183908-e1648e67c298 // indirect
	github.com/stmcginnis/gofish v0.15.1-0.20231121142100-22a60a77be91 // indirect
	github.com/stolostron/cluster-lifecycle-api v0.0.0-20240109072430-f5fe6043d1f8 // indirect
	github.com/stolostron/klusterlet-addon-controller v0.0.0-20240606130554-01338045271a // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	knative.dev/pkg v0.0.0-20240423132823-3c6badc82748 // indirect
	maistra.io/api v0.0.0-20230704084350-dfc96815fb16 // indirect
	open-cluster-management.io/api v0.13.0 // indirect
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/argoproj-labs/argocd-operator v0.10.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.50.25 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/clarketm/json v1.17.1 // indirect
	github.com/containernetworking/cni v1.2.1-0.20240513144334-1e7858f9879a // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-json v0.0.0-20230131223807-18775e0fb4fb // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/ign-converter v0.0.0-20230417193809-cee89ea7d8ff // indirect
	github.com/coreos/ignition v0.35.0 // indirect
	github.com/coreos/ignition/v2 v2.18.0 // indirect
	github.com/coreos/vcontext v0.0.0-20231102161604-685dc7299dc5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.12.0 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20220118164431-d8423dcdf344 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/analysis v0.21.4 // indirect
	github.com/go-openapi/errors v0.20.3 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.21.2 // indirect
	github.com/go-openapi/spec v0.20.7 // indirect
	github.com/go-openapi/strfmt v0.21.3 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.22.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/cel-go v0.18.2 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240424215950-a892ee059fd6 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana-operator/grafana-operator/v4 v4.10.1 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/h2non/filetype v1.1.3 // indirect
	github.com/h2non/go-is-svg v0.0.0-20160927212452-35e8c4b0612c // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.5 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.8 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/hashicorp/vault/api v1.13.0 // indirect
	github.com/hashicorp/vault/api/auth/approle v0.5.0 // indirect
	github.com/hashicorp/vault/api/auth/kubernetes v0.5.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kube-object-storage/lib-bucket-provisioner v0.0.0-20221122204822-d1a8c34382f1 // indirect
	github.com/libopenstorage/secrets v0.0.0-20231011182615-5f4b25ceede1 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/noobaa/noobaa-operator/v5 v5.0.0-20231213124549-5d7b0417716d // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/openshift-kni/cluster-group-upgrades-operator v0.0.0-20240423171335-f07cdbf8af2c
	github.com/openshift/assisted-service v1.0.10-0.20230830164851-6573b5d7021d // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/openshift/library-go v0.0.0-20240419113445-f1541d628746 // indirect
	github.com/openshift/local-storage-operator v0.0.0-20240422172451-2a80d7f6681d // indirect
	github.com/openshift/ptp-operator v0.0.0-20240404165119-29a3d7b3d60b // indirect
	github.com/operator-framework/operator-lifecycle-manager v0.28.0 // indirect
	github.com/operator-framework/operator-registry v1.41.0 // indirect
	github.com/otiai10/copy v1.14.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/red-hat-storage/ocs-operator v0.4.13
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rook/rook/pkg/apis v0.0.0-20231215165123-32de0fb5f69b // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/samber/lo v1.39.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/thoas/go-funk v0.9.2 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.mongodb.org/mongo-driver v1.15.0 // indirect
	go.starlark.net v0.0.0-20240123142251-f86470692795 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/crypto v0.24.0
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/oauth2 v0.20.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/term v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240415180920-8c6c420018be // indirect
	google.golang.org/grpc v1.63.2 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/gorm v1.24.5 // indirect
	k8s.io/apiextensions-apiserver v0.29.4 // indirect
	k8s.io/apiserver v0.29.4 // indirect
	k8s.io/cli-runtime v0.29.4 // indirect
	k8s.io/component-base v0.29.4 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-aggregator v0.29.4 // indirect
	k8s.io/kube-openapi v0.0.0-20240411171206-dc4e619f62f3 // indirect
	k8s.io/kubectl v0.29.3 // indirect
	k8s.io/kubelet v0.29.4
	sigs.k8s.io/container-object-storage-interface-api v0.1.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.6-0.20230721195810-5c8923c5ff96 // indirect
	sigs.k8s.io/kustomize/api v0.17.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.17.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace (
	github.com/imdario/mergo => github.com/imdario/mergo v0.3.16
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20240508132640-2b61056c9758 // release-4.16
	github.com/kubernetes-incubator/external-storage => github.com/libopenstorage/external-storage v1.8.1-0.20211117215743-47ffded9c408
	github.com/metal3-io/baremetal-operator/pkg/hardwareutils => github.com/metal3-io/baremetal-operator/pkg/hardwareutils v0.6.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20240605153344-636e2c17106f // release-4.16
	github.com/openshift/assisted-service/api => github.com/openshift/assisted-service/api v0.0.0-20240529165317-6b26a25e2ae7 // release-4.16
	github.com/openshift/assisted-service/models => github.com/openshift/assisted-service/models v0.0.0-20240529165317-6b26a25e2ae7 // release-4.16
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.1
	github.com/openshift/installer => github.com/openshift/installer v0.9.0-master.0.20230306121016-3485fddca1c3 // master
	github.com/portworx/sched-ops => github.com/portworx/sched-ops v1.20.4-rc1
	k8s.io/api => k8s.io/api v0.29.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.29.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.29.4
	k8s.io/apiserver => k8s.io/apiserver v0.29.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.29.4
	k8s.io/client-go => k8s.io/client-go v0.29.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.29.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.29.4
	k8s.io/code-generator => k8s.io/code-generator v0.29.4
	k8s.io/component-base => k8s.io/component-base v0.29.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.29.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.29.4
	k8s.io/cri-api => k8s.io/cri-api v0.29.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.29.4
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.29.4
	k8s.io/endpointslice => k8s.io/endpointslice v0.29.4
	k8s.io/kms => k8s.io/kms v0.29.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.29.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.29.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.29.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.29.4
	k8s.io/kubectl => k8s.io/kubectl v0.29.4
	k8s.io/kubelet => k8s.io/kubelet v0.29.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.29.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.29.4
	k8s.io/metrics => k8s.io/metrics v0.29.4
	k8s.io/mount-utils => k8s.io/mount-utils v0.29.4
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.29.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.29.4
)

exclude github.com/kubernetes-incubator/external-storage v0.20.4-openstorage-rc2
