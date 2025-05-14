// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package v1alpha4

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

// VirtualMachineAffinitySpec defines the group of affinity scheduling rules.
type VirtualMachineAffinitySpec struct {
	// +optional

	// ZoneAffinity describes affinity scheduling rules related to a zone.
	ZoneAffinity *VirtualMachineAffinityZoneAffinitySpec `json:"zoneAffinity,omitempty"`
}
