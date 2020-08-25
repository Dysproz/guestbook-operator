package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GuestbookSpec defines the desired state of Guestbook
type GuestbookSpec struct {
	RedisSize        int    `json:"redisSize,omitempty"`
	GuestbookSize    int    `json:"guestbookSize,omitempty"`
	RedisMasterImage string `json:"redisMasterImage,omitempty"`
	RedisSlaveImage  string `json:"redisSlaveImage,omitempty"`
	GuestbookImage   string `json:"guestbookImage,omitempty"`
}

// GuestbookStatus defines the observed state of Guestbook
type GuestbookStatus struct {
	Ready bool `json:"ready,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Guestbook is the Schema for the guestbooks API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=guestbooks,scope=Namespaced
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Is Guestbook ready?"
type Guestbook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GuestbookSpec   `json:"spec,omitempty"`
	Status GuestbookStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GuestbookList contains a list of Guestbook
type GuestbookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Guestbook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Guestbook{}, &GuestbookList{})
}
