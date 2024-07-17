module github.com/openshift-kni/eco-gotests

go 1.20

require (
	github.com/Juniper/go-netconf v0.3.0
	github.com/NVIDIA/gpu-operator v1.11.1
	github.com/cavaliergopher/cpio v1.0.1
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/golang/glog v1.1.2
	github.com/hashicorp/go-version v1.6.0
	github.com/k8snetworkplumbingwg/multi-networkpolicy v0.0.0-20230301165931-f1873dc329c6
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.4.0
	github.com/k8snetworkplumbingwg/sriov-network-operator v0.0.0-20201204053545-49045c36efb9
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/metal3-io/baremetal-operator/apis v0.4.0
	github.com/metallb/metallb-operator v0.0.0-20231019113803-fbc3c30718ef
	github.com/nmstate/kubernetes-nmstate/api v0.0.0-20231116153922-80c6e01df02e
	github.com/onsi/ginkgo/v2 v2.13.0
	github.com/onsi/gomega v1.28.0
	github.com/openshift-kni/eco-goinfra v0.0.0-20240715131405-c1d20ad6ec42
	github.com/openshift-kni/k8sreporter v1.0.5
	github.com/openshift/api v3.9.1-0.20190916204813-cdbe64fb0c91+incompatible
	github.com/openshift/assisted-service/api v0.0.0
	github.com/openshift/assisted-service/models v0.0.0
	github.com/openshift/cluster-nfd-operator v0.0.0-20231114113211-9c86f35b4220
	github.com/openshift/cluster-node-tuning-operator v0.0.0-20230704170229-287fdce04769
	github.com/openshift/hive/apis v0.0.0-20220222213051-def9088fdb5a
	github.com/openshift/installer v0.16.1
	github.com/openshift/machine-config-operator v0.0.1-0.20230807154212-886c5c3fc7a9
	github.com/operator-framework/api v0.18.0
	github.com/rh-ecosystem-edge/kernel-module-management v0.0.0-20231116085833-96c2cd1e77e9
	github.com/wk8/go-ordered-map/v2 v2.1.8
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63
	gopkg.in/k8snetworkplumbingwg/multus-cni.v4 v4.0.2
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.27.7
	k8s.io/apimachinery v0.27.7
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.15.3
)

require (
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/coreos/go-json v0.0.0-20230131223807-18775e0fb4fb // indirect
	github.com/go-openapi/analysis v0.21.4 // indirect
	github.com/go-openapi/errors v0.20.3 // indirect
	github.com/go-openapi/loads v0.21.2 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/strfmt v0.21.3 // indirect
	github.com/go-openapi/validate v0.22.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.4 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	go.mongodb.org/mongo-driver v1.11.2 // indirect
	gorm.io/gorm v1.24.5 // indirect
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/PaesslerAG/jsonpath v0.1.1 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20200220154158-5ae607161559 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230305170008-8188dc5388df // indirect
	github.com/argoproj-labs/argocd-operator v0.7.0 // indirect
	github.com/aws/aws-sdk-go v1.44.215 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/bugsnag/panicwrap v1.3.4 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/clarketm/json v1.17.1 // indirect
	github.com/containernetworking/cni v1.1.2 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/coreos/ign-converter v0.0.0-20230417193809-cee89ea7d8ff // indirect
	github.com/coreos/ignition v0.35.0 // indirect
	github.com/coreos/ignition/v2 v2.15.0 // indirect
	github.com/coreos/vcontext v0.0.0-20230201181013-d72178a18687 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/zapr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/cel-go v0.15.3 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20230502171905-255e3b9b56de // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/grafana-operator/grafana-operator/v4 v4.10.1 // indirect
	github.com/h2non/filetype v1.1.1 // indirect
	github.com/h2non/go-is-svg v0.0.0-20160927212452-35e8c4b0612c // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kdomanski/iso9660 v0.2.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/metal3-io/baremetal-operator/pkg/hardwareutils v0.2.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nutanix-cloud-native/prism-go-client v0.2.1-0.20220804130801-c8a253627c64 // indirect
	github.com/openshift-kni/cluster-group-upgrades-operator v0.0.0-20230810161322-3ef5efb0de85 // indirect
	github.com/openshift/assisted-service v1.0.10-0.20230830164851-6573b5d7021d // indirect
	github.com/openshift/client-go v0.0.0-20230807132528-be5346fb33cb // indirect
	github.com/openshift/custom-resource-status v1.1.3-0.20220503160415-f2fdb4999d87 // indirect
	github.com/openshift/library-go v0.0.0-20231020125025-211b32f1a1f2 // indirect
	github.com/openshift/local-storage-operator v0.0.0-20230904072424-14e8d821fe80 // indirect
	github.com/openshift/ptp-operator v0.0.0-20230608145834-0f37b622bc3b // indirect
	github.com/operator-framework/operator-lifecycle-manager v0.26.0 // indirect
	github.com/operator-framework/operator-registry v1.30.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.17.0 // indirect
	github.com/prometheus/client_model v0.4.1-0.20230718164431-9a2bf3000d16 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/thoas/go-funk v0.9.3 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	go4.org v0.0.0-20201209231011-d4a079459e60 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/oauth2 v0.12.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.12.1-0.20230815132531-74c255bcf846 // indirect
	gomodules.xyz/jsonpatch/v2 v2.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.27.7 // indirect
	k8s.io/apiserver v0.27.7 // indirect
	k8s.io/component-base v0.27.7 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-aggregator v0.27.4 // indirect
	k8s.io/kube-openapi v0.0.0-20230515203736-54b630e78af5 // indirect
	k8s.io/kubelet v0.27.7 // indirect
	k8s.io/kubernetes v1.26.1 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
	open-cluster-management.io/governance-policy-propagator v0.12.0 // indirect
	open-cluster-management.io/multicloud-operators-subscription v0.11.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kube-storage-version-migrator v0.0.6-0.20230721195810-5c8923c5ff96 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.3.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20240311224155-7a362a543c29 // release-4.14
	github.com/openshift/api => github.com/openshift/api v0.0.0-20231117205818-971e4ba78c9a // release-4.14
	github.com/openshift/assisted-service/api => github.com/openshift/assisted-service/api v0.0.0-20230907160625-4e1a1e59cef0 // release-4.14
	github.com/openshift/assisted-service/models => github.com/openshift/assisted-service/models v0.0.0-20230907160625-4e1a1e59cef0 // release-4.14
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20230807132528-be5346fb33cb // release-4.14
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20231114114555-1e657ecbea8f // relase-4.14
	github.com/openshift/installer => github.com/openshift/installer v0.0.0-20230725173033-8cd8f79a20b3 // release-4.14
	github.com/openshift/ptp-operator => github.com/openshift/ptp-operator v0.0.0-20231012160558-ef5f19567818
	k8s.io/api => k8s.io/api v0.27.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.27.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.27.4
	k8s.io/apiserver => k8s.io/apiserver v0.27.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.27.4
	k8s.io/client-go => k8s.io/client-go v0.27.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.27.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.27.4
	k8s.io/code-generator => k8s.io/code-generator v0.27.4
	k8s.io/component-base => k8s.io/component-base v0.27.4
	k8s.io/component-helpers => k8s.io/component-helpers v0.27.4
	k8s.io/controller-manager => k8s.io/controller-manager v0.27.4
	k8s.io/cri-api => k8s.io/cri-api v0.27.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.27.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.27.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.27.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.27.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.27.4
	k8s.io/kubectl => k8s.io/kubectl v0.27.4
	k8s.io/kubelet => k8s.io/kubelet v0.27.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.27.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.27.4
	k8s.io/metrics => k8s.io/metrics v0.27.4
	k8s.io/mount-utils => k8s.io/mount-utils v0.27.4
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.27.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.27.4
)
