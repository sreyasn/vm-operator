package virtualmachine

import (
	"fmt"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha4"
	pkgctx "github.com/vmware-tanzu/vm-operator/pkg/context"
	"github.com/vmware-tanzu/vm-operator/pkg/util/ptr"
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
	snap, _ := vm.FindSnapshot(args.VMCtx, obj.Name)
	if snap != nil {
		// update vm.status with currentSnapshot
		updateVMStatusCurrentSnapshot(args.VMCtx, obj)
		// return early, snapshot found
		return nil
	}

	// if no snapshot was found, create it
	err := createSnapshot(args.VMCtx, vm, obj.Name, obj.Spec.Description, obj.Spec.Memory, obj.Spec.QuiesceSpec)
	if err != nil {
		args.VMCtx.Logger.Error(err, "failed to create snapshot for VM", "snapshot", obj.Name)
		return err
	}

	// update vm.status with currentSnapshot
	updateVMStatusCurrentSnapshot(args.VMCtx, obj)
	return nil
}

func createSnapshot(vmCtx pkgctx.VirtualMachineContext, vcVM *object.VirtualMachine, name string, description string,
	memory *bool, quiesce *vmopv1.QuiesceSpec) error {
	var quiesceSpec *types.VirtualMachineGuestQuiesceSpec
	if quiesce != nil {
		quiesceSpec = &types.VirtualMachineGuestQuiesceSpec{
			Timeout: int32(quiesce.Timeout.Round(time.Minute).Minutes()),
		}
	}

	snapMemory := false
	if memory != nil {
		snapMemory = *memory
	}

	t, err := vcVM.CreateSnapshotEx(vmCtx, name, description, snapMemory, quiesceSpec)
	if err != nil {
		return err
	}

	// wait for task to finish
	taskInfo, err := t.WaitForResult(vmCtx)
	if err != nil {
		if taskInfo != nil {
			vmCtx.Logger.V(5).Error(err, "create snapshot task failed", "taskInfo", taskInfo)
		}
	}

	_, ok := taskInfo.Result.(types.ManagedObjectReference)
	if !ok {
		return fmt.Errorf("create snapshot VM task failed: %w", err)
	}

	return nil
}

func updateVMStatusCurrentSnapshot(vmCtx pkgctx.VirtualMachineContext, obj vmopv1.VirtualMachineSnapshot) {
	vmCtx.VM.Status.CurrentSnapshot = &corev1.TypedLocalObjectReference{
		APIGroup: ptr.To(vmopv1.GroupName),
		Kind:     obj.Kind,
		Name:     obj.Name,
	}
}
