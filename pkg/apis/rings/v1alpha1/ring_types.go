package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type RingPort struct {
	// The name of this port within the service. This must be a DNS_LABEL.
	// All ports within a ServiceSpec must have unique names. This maps to
	// the 'Name' field in EndpointPort objects.
	// Optional if only one ServicePort is defined on this service.
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
	// Default is TCP.
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty" protobuf:"bytes,2,opt,name=protocol,casttype=Protocol"`

	// The port that will be exposed by this service.
	Port int32 `json:"port" protobuf:"varint,3,opt,name=port"`

	// Number or name of the port to access on the pods targeted by the service.
	// Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
	// If this is a string, it will be looked up as a named port in the
	// target Pod's container ports. If this is not specified, the value
	// of the 'port' field is used (an identity map).
	// This field is ignored for services with clusterIP=None, and should be
	// omitted or set equal to the 'port' field.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service
	// +optional
	TargetPort intstr.IntOrString `json:"targetPort,omitempty" protobuf:"bytes,4,opt,name=targetPort"`
}

type RingGroup struct {
	// The name of the group to be included in the ring
	Name string `json:"name"`

	// The initial users to be included in the group
	// +optional
	InitialUsers []string `json:"initialUsers,omitempty"`
}

type RingRouting struct {
	// The target group of the ring
	Group RingGroup `json:"group"`
	// Service will target the deployments with this service tag
	Service string `json:"service"`
	// Version will target the deployments with this major version tag
	Version string `json:"version"`
	// Branch will target the deployments with this branch tag
	Branch string `json:"branch"`
	// Ports will expose these ports on the services and verified against the Deployment found
	Ports []RingPort `json:"ports"`
}

// RingSpec defines the desired state of Ring
// +k8s:openapi-gen=true
type RingSpec struct {
	// Deploy marks whether this ring will be deployed to the live environment
	Deploy bool `json:"deploy"`
	// Routing describes the service, group and users to be included in the ring
	Routing RingRouting `json:"routing"`
}

// RingStatus defines the observed state of Ring
// +k8s:openapi-gen=true
type RingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Ring is the Schema for the rings API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Ring struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RingSpec   `json:"spec,omitempty"`
	Status RingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RingList contains a list of Ring
type RingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ring `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ring{}, &RingList{})
}
