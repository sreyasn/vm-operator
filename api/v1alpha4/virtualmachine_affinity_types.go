// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package v1alpha4

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:validation:Enum=In;NotIn;Exists;DoesNotExist;Gt;Lt

// ZoneSelectorOperator specifies the type of operator used by
// the zone selector to represent key-value relationships.
type ZoneSelectorOperator string

const (
	ZoneSelectorOpIn           ZoneSelectorOperator = "In"
	ZoneSelectorOpNotIn        ZoneSelectorOperator = "NotIn"
	ZoneSelectorOpExists       ZoneSelectorOperator = "Exists"
	ZoneSelectorOpDoesNotExist ZoneSelectorOperator = "DoesNotExist"
	ZoneSelectorOpGt           ZoneSelectorOperator = "Gt"
	ZoneSelectorOpLt           ZoneSelectorOperator = "Lt"
)

// ZoneSelectorRequirement defines the key value relationships for a matching zone selector.
type ZoneSelectorRequirement struct {
	// Key is the label key to which the selector applies.
	Key string `json:"key"`

	// Operator represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator ZoneSelectorOperator `json:"operator"`

	// +optional
	// +listType=atomic

	// Values is a list of values to which the operator applies.
	// If the operator is In or NotIn, the values list must be non-empty.
	// If the operator is Exists or DoesNotExist, the values list must be empty.
	// If the operator is Gt or Lt, the values list must have a single element,
	// which will be interpreted as an integer.
	Values []string `json:"values,omitempty"`
}

// ZoneSelectorTerm defines the matching zone selector requirements for zone based affinity scheduling.
type ZoneSelectorTerm struct {
	// +optional
	// +listType=atomic

	// MatchExpressions is a list of zone selector requirements by zone's
	// labels.
	MatchExpressions []ZoneSelectorRequirement `json:"matchExpressions,omitempty"`

	// +optional
	// +listType=atomic

	// MatchFields is a list of zone selector requirements by zone's fields.
	MatchFields []ZoneSelectorRequirement `json:"matchFields,omitempty"`
}

// VirtualMachineAffinityZoneAffinitySpec defines the affinity scheduling rules
// related to zones.
type VirtualMachineAffinityZoneAffinitySpec struct {
	// +listType=atomic

	// ZoneSelectorTerms is a list of zone selector terms. The terms are ORed.
	ZoneSelectorTerms []ZoneSelectorTerm `json:"zoneSelectorTerms"`
}

// VMAffinityTerm defines the VM affinity/anti-affinity term.
type VMAffinityTerm struct {
	// +optional

	// LabelSelector is a label query over a set of VMs.
	// When omitted, this term matches with no VMs.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// TopologyKey describes where this VM should be co-located (affinity) or not
	// co-located (anti-affinity). The value is used to match a label on a node, i.e. when
	// set to topology.kubernetes.io/zone, it means this term applies to nodes
	// with that label present.
	TopologyKey string `json:"topologyKey"`
}

// WeightedVMAffinityTerm defines the weighted VM affinity/anti-affinity term.
type WeightedVMAffinityTerm struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100

	// Weight describes the weight of the associated term.
	Weight int32 `json:"weight"`

	// VMAffinityTerm describes the affinity term.
	VMAffinityTerm VMAffinityTerm `json:"vmAffinityTerm"`
}

// VirtualMachineAffinityVMAffinitySpec defines the affinity requirements for scheduling
// rules related to other VMs.
type VirtualMachineAffinityVMAffinitySpec struct {
	// +optional
	// +listType=atomic

	// RequiredDuringSchedulingIgnoredDuringExecution describes affinity
	// requirements that must be met or the VM will not be scheduled.
	//
	// When there are multiple elements, the lists of nodes corresponding to
	// each term are intersected, i.e. all terms must be satisfied.
	RequiredDuringSchedulingIgnoredDuringExecution []VMAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	// +optional
	// +listType=atomic

	// PreferredDuringSchedulingIgnoredDuringExecution describes affinity
	// requirements that should be met, but the VM can still be scheduled if
	// the requirement cannot be satisfied. The scheduler will prefer to schedule VMs to nodes
	// that satisfy the anti-affinity expressions specified by this field, but it may choose a node that
	// violates one or more of the expressions. The node that is most preferred is the one with the
	// greatest sum of weights.
	//
	// When there are multiple elements, the lists of nodes corresponding to
	// each term are intersected, i.e. all terms must be satisfied.
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedVMAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// VirtualMachineAntiAffinityVMAffinitySpec defines the anti-affinity requirements for scheduling
// rules related to other VMs.
type VirtualMachineAntiAffinityVMAffinitySpec struct {
	// +optional
	// +listType=atomic

	// RequiredDuringSchedulingIgnoredDuringExecution describes anti-affinity
	// requirements that must be met or the VM will not be scheduled.
	//
	// When there are multiple elements, the lists of nodes corresponding to
	// each term are intersected, i.e. all terms must be satisfied.
	RequiredDuringSchedulingIgnoredDuringExecution []VMAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	// +optional
	// +listType=atomic

	// PreferredDuringSchedulingIgnoredDuringExecution describes anti-affinity
	// requirements that should be met, but the VM can still be scheduled if
	// the requirement cannot be satisfied. The scheduler will prefer to schedule VMs to nodes
	// that satisfy the affinity expressions specified by this field, but it may choose a node that
	// violates one or more of the expressions. The node that is most preferred is the one with the
	// greatest sum of weights.
	//
	// When there are multiple elements, the lists of nodes corresponding to
	// each term are intersected, i.e. all terms must be satisfied.
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedVMAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// VirtualMachineAffinitySpec defines the group of affinity scheduling rules.
type VirtualMachineAffinitySpec struct {
	// +optional

	// ZoneAffinity describes affinity scheduling rules related to a zone.
	ZoneAffinity *VirtualMachineAffinityZoneAffinitySpec `json:"zoneAffinity,omitempty"`

	// +optional

	// VMAffinity describes affinity scheduling rules related to other VMs.
	VMAffinity *VirtualMachineAffinityVMAffinitySpec `json:"vmAffinity,omitempty"`

	// +optional

	// VMAntiAffinity describes anti-affinity scheduling rules related to other VMs.
	VMAntiAffinity *VirtualMachineAntiAffinityVMAffinitySpec `json:"vmAntiAffinity,omitempty"`
}
