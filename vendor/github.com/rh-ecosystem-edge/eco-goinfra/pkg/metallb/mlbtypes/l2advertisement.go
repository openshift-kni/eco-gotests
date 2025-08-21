package mlbtypes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// L2AdvertisementSpec defines the desired state of L2Advertisement.
type L2AdvertisementSpec struct {
	// The list of IPAddressPools to advertise via this advertisement, selected by name.
	// +optional
	IPAddressPools []string `json:"ipAddressPools,omitempty"`
	// A selector for the IPAddressPools which would get advertised via this advertisement.
	// If no IPAddressPool is selected by this or by the list, the advertisement is applied to all the IPAddressPools.
	// +optional
	IPAddressPoolSelectors []metav1.LabelSelector `json:"ipAddressPoolSelectors,omitempty"`
	// NodeSelectors allows to limit the nodes to announce as next hops for the LoadBalancer IP.
	// When empty, all the nodes having  are announced as next hops.
	// +optional
	NodeSelectors []metav1.LabelSelector `json:"nodeSelectors,omitempty"`
	// A list of interfaces to announce from. The LB IP will be announced only from these interfaces.
	// If the field is not set, we advertise from all the interfaces on the host.
	// +optional
	Interfaces []string `json:"interfaces,omitempty"`
}

// L2AdvertisementStatus defines the observed state of L2Advertisement.
type L2AdvertisementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// L2Advertisement allows to advertise the LoadBalancer IPs provided
// by the selected pools via L2.
type L2Advertisement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   L2AdvertisementSpec   `json:"spec,omitempty"`
	Status L2AdvertisementStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// L2AdvertisementList contains a list of L2Advertisement.
type L2AdvertisementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L2Advertisement `json:"items"`
}
