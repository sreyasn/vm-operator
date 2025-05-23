package virtualmachine

import (
	"fmt"

	vmopv1common "github.com/vmware-tanzu/vm-operator/api/v1alpha4/common"
	"github.com/vmware/govmomi/vim25/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha4"
	pkgctx "github.com/vmware-tanzu/vm-operator/pkg/context"
	"github.com/vmware/govmomi/object"
)

// SnapshotArgs contains the options for createSnapshot.
type SnapshotArgs struct {
	VMCtx      pkgctx.VirtualMachineContext
	VcVM       *object.VirtualMachine
	VMSnapshot vmopv1.VirtualMachineSnapshot
	K8sClient  client.Client
}

func SnapshotVirtualMachine(args SnapshotArgs) error {
	obj := args.VMSnapshot
	vm := args.VcVM

	// find snapshot by name
	snap, err := vm.FindSnapshot(args.VMCtx, obj.Name)
	if snap != nil {

		// update vm.status with currentSnapshot
		updateVMStatusCurrentSnapshot(args.VMCtx, obj)
		// patch the snapShot status
		patchSnapshotStatus(args.VMCtx, args.K8sClient, obj, true)
		// return early, snapshot found
		return nil
	}

	// if no snapshot was found, create it
	snap, err = createSnapshot(args.VMCtx, vm, obj.Name, obj.Spec.Description, obj.Spec.Memory, obj.Spec.QuiesceSpec)
	if err != nil {
		args.VMCtx.Logger.Error(err, "failed to create snapshot for VM")
		patchSnapshotStatus(args.VMCtx, args.K8sClient, obj, false)
		return err
	}

	// update vm.status with currentSnapshot
	updateVMStatusCurrentSnapshot(args.VMCtx, obj)
	if err := patchSnapshotStatus(args.VMCtx, args.K8sClient, obj, true); err != nil {
		return nil
	}

	return nil
}

func createSnapshot(vmCtx pkgctx.VirtualMachineContext, vcVM *object.VirtualMachine, name string, description string,
	memory *bool, quiesceSpec *vmopv1.QuiesceSpec) (*types.ManagedObjectReference, error) {

	quiesed := false
	if quiesceSpec != nil {
		quiesed = true
	}

	memoryFlag := false
	if memory != nil {
		memoryFlag = true
	}

	t, err := vcVM.CreateSnapshot(vmCtx, name, description, memoryFlag, quiesed)
	if err != nil {
		return nil, err
	}

	// wait for task to finish
	taskInfo, err := t.WaitForResult(vmCtx)
	if err != nil {
		if taskInfo != nil {
			vmCtx.Logger.V(5).Error(err, "create snapshot task failed", "taskInfo", taskInfo)
		}
	}

	snapRef, ok := taskInfo.Result.(types.ManagedObjectReference)
	if !ok {
		return nil, fmt.Errorf("create snapshot VM task failed: %w", err)
	}

	return &snapRef, nil
}

func updateVMStatusCurrentSnapshot(vmCtx pkgctx.VirtualMachineContext, obj vmopv1.VirtualMachineSnapshot) {
	vmCtx.VM.Status.CurrentSnapshot = &vmopv1common.LocalObjectRef{
		APIVersion: obj.APIVersion,
		Kind:       obj.Kind,
		Name:       obj.Name,
	}
}

func patchSnapshotStatus(vmCtx pkgctx.VirtualMachineContext, k8sClient client.Client,
	obj vmopv1.VirtualMachineSnapshot, success bool) error {

	snapShot := &vmopv1.VirtualMachineSnapshot{}
	objKey := client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}
	// get snapshot again to ensure it's up-to-date.
	err := k8sClient.Get(vmCtx, objKey, snapShot)
	if err != nil {
		vmCtx.Logger.Error(err, "failed to get snapshot resource", "snapshot", objKey)
		return err
	}

	snapPatch := client.MergeFrom(snapShot.DeepCopy())
	if !success {
		failedPhase := vmopv1.VMSnapshotFailed
		snapShot.Status.Phase = &failedPhase

	} else {
		successPhase := vmopv1.VMSnapshotSucceeded
		snapShot.Status.Phase = &successPhase
	}

	if err := k8sClient.Status().Patch(vmCtx, snapShot, snapPatch); err != nil {
		return fmt.Errorf(
			"failed to patch snapshot status resource %s: err: %s", objKey, err.Error())
	}

	return nil
}
