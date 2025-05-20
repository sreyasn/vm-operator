package virtualmachine

import (
	"fmt"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha4"
	pkgctx "github.com/vmware-tanzu/vm-operator/pkg/context"
	"github.com/vmware/govmomi/object"
)

// SnapshotArgs contains the options for createSnapshot.
type SnapshotArgs struct {
	VMCtx      pkgctx.VirtualMachineContext
	VcVM       *object.VirtualMachine
	VMSnapshot vmopv1.VirtualMachineSnapshot
}

func SnapshotVirtualMachine(args SnapshotArgs) error {
	obj := args.VMSnapshot
	// find snapshot by name
	vm := args.VcVM
	snap, _ := vm.FindSnapshot(args.VMCtx, obj.Name)
	// if snapshot is not found, create it
	if snap == nil {
		if err := createSnapshot(args.VMCtx, vm, obj.Name, obj.Spec.Description, obj.Spec.Memory, obj.Spec.QuiesceSpec); err != nil {
			args.VMCtx.Logger.Error(err, "failed to create snapshot for VM")
			return err
		}
	}

	// set status on VM
	// patch the VM Snapshot resource

	return nil
}

func createSnapshot(vmCtx pkgctx.VirtualMachineContext, vcVM *object.VirtualMachine, name string, description string,
	memory *bool, quiesceSpec *vmopv1.QuiesceSpec) error {

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
		return err
	}

	// wait for task to finish
	if taskInfo, err := t.WaitForResult(vmCtx); err != nil {
		if taskInfo != nil {
			vmCtx.Logger.V(5).Error(err, "create snapshot task failed", "taskInfo", taskInfo)
		}
		return fmt.Errorf("create snapshot VM task failed: %w", err)
	}

	return nil
}
