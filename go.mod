module github.com/openshift-kni/eco-gotests

go 1.18

require (
	github.com/golang/glog v1.0.0
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.1-0.20210510153419-66a699ae3b05
	github.com/k8snetworkplumbingwg/sriov-network-operator v0.0.0-20201204053545-49045c36efb9
	github.com/metallb/metallb-operator v0.10.2
	github.com/openshift-kni/numaresources-operator v0.4.11-0rc0.0.20220727175518-37b5d067b623
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20211209144617-7385dd6338e3
	github.com/openshift/machine-config-operator v0.0.1-0.20211105081319-76d6155c1dab
	github.com/openshift/ptp-operator v0.0.0-20220616154021-73c849ad6b4b
	github.com/operator-framework/operator-lifecycle-manager v0.22.0
	github.com/smart-edge-open/sriov-fec-operator/sriov-fec v0.0.0-20220421094756-1803088bf003
	go.universe.tf/metallb v0.1.1-0.20220422145807-099c8990dc55
	k8s.io/api v0.24.3
	k8s.io/apiextensions-apiserver v0.24.0
	k8s.io/apimachinery v0.24.3
	k8s.io/client-go v1.5.2
	sigs.k8s.io/controller-runtime v0.12.1
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/ajeddeloh/go-json v0.0.0-20170920214419-6a2fe990e083 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/clarketm/json v1.14.1 // indirect
	github.com/coreos/fcct v0.5.0 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/coreos/ign-converter v0.0.0-20201123214124-8dac862888aa // indirect
	github.com/coreos/ignition v0.35.0 // indirect
	github.com/coreos/ignition/v2 v2.7.0 // indirect
	github.com/coreos/vcontext v0.0.0-20191017033345-260217907eb5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/gomega v1.19.0 // indirect
	github.com/operator-framework/api v0.16.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb // indirect
	go4.org v0.0.0-20200104003542-c7e774b10ea0 // indirect
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sys v0.0.0-20220319134239-a9b59b0215f8 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/component-base v0.24.3 // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/k8snetworkplumbingwg/sriov-network-operator => github.com/openshift/sriov-network-operator v0.0.0-20220623160955-488941958967 // release-4.7
	github.com/openshift/api => github.com/openshift/api v0.0.0-20220525145417-ee5b62754c68 // release-4.11
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20211209144617-7385dd6338e3 // release-4.7
	k8s.io/api => k8s.io/api v0.24.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.3
	k8s.io/apiserver => k8s.io/apiserver v0.24.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.24.3
	k8s.io/client-go => k8s.io/client-go v0.24.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.3
	k8s.io/code-generator => k8s.io/code-generator v0.24.3
	k8s.io/component-base => k8s.io/component-base v0.24.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.24.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.3
	k8s.io/cri-api => k8s.io/cri-api v0.24.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.3
	k8s.io/kubectl => k8s.io/kubectl v0.24.3
	k8s.io/kubelet => k8s.io/kubelet v0.24.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.3
	k8s.io/metrics => k8s.io/metrics v0.24.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.3
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.24.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.3
)
