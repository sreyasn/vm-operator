// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package v1alpha4

import (
	"time"

	vmopv1common "github.com/vmware-tanzu/vm-operator/api/v1alpha4/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VMSnapshotPhase represents the phase of a VM snapshot.
type VMSnapshotPhase string

const (
	// VM snapshot operation is successful.
	VMSnapshotSucceeded VMSnapshotPhase = "Succeeded"
	// VM snapshot operation is in progress.
	VMSnapshotInProgress VMSnapshotPhase = "In Progress"
	// VM snapshot operation has failed.
	VMSnapshotFailed VMSnapshotPhase = "Failed"
)

const (
	// VirtualMachineSnapshotReadyCondition represents the condition
	// that the virtual machine snapshot is ready.
	VirtualMachineSnapshotReadyCondition = "VirtualMachineSnapshotReady"
)

// QuiesceSpec represents specifications that will be used to quiesce
// the guest when taking a snapshot.
type QuiesceSpec struct {
	Timeout *time.Duration `json:"timeout,omitempty"`

	// TODO: Windows specific quiesce details from
	// https://opengrok2.vdp.lvn.broadcom.net/xref/main.perforce.1666/bora/vim/vmodl/vim/vm/WindowsQuiesceSpec.java?r=7214958
}

// VirtualMachineSnapshotSpec defines the desired state of VirtualMachineSnapshot.
type VirtualMachineSnapshotSpec struct {
	// +optional
	// +kubebuilder:default=false
	//
	// Memory represents whether the snapshot includes the VM's
	// memory. If true, a dump of the internal state of the virtual
	// machine (a memory dump) is included in the snapshot. Memory
	// snapshots consume time and resources and thus, take longer to
	// create.
	// The virtual machine must support this capability.
	// When set to false, the power state of the snapshot is set to
	// false.
	// For a VM in suspended state, memory is always included
	// in the snashot.
	Memory *bool `json:"memory,omitempty"`

	// +optional
	// +kubebuilder:default=false

	// Quiesce represents the spec used for granular control over quiesce details.
	// If quiesceSpec is set and the virtual machine is powered on when the
	// snapshot is taken, VMware Tools is used to quiesce the file
	// system in the virtual machine. This assures that a disk snapshot
	// represents a consistent state of the guest file systems. If the virtual
	// machine is powered off or VMware Tools are not available, the quiesce
	// spec is ignored.
	QuiesceSpec *QuiesceSpec `json:"quiesce,omitempty"`

	// +optional
	//
	// Description represents a description of the snapshot.
	Description string `json:"description,omitempty"`

	// Source represents the source VM for which the Snapshot is
	// requested.
	Source *vmopv1common.LocalObjectRef `json:"source:omitempty"`
}

// VirtualMachineSnapshotStatus defines the observed state of VirtualMachineSnapshot.
type VirtualMachineSnapshotStatus struct {
	// +optional

	// Phase represents the phase of a VM snapshot operation.
	Phase *VMSnapshotPhase `json:"phase:omitempty"`

	// +optional

	// Children represents the snapshots for which this snapshot is
	// the parent.
	Children []*vmopv1common.LocalObjectRef `json:"children,omitempty"`

	// +optional

	// Conditions describes the observed conditions of the VirtualMachine.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=vmsnapshot
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

// VirtualMachineSnapshot is the schema for the virtualmachinesnapshot API.
type VirtualMachineSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSnapshotSpec   `json:"spec,omitempty"`
	Status VirtualMachineSnapshotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VirtualMachineSnapshotList contains a list of VirtualMachineSnapshot.
type VirtualMachineSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineSnapshot `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &VirtualMachineSnapshot{}, &VirtualMachineSnapshotList{})
}
