package virtualmachinesnapshot

import (
	"context"
	"errors"
	"fmt"
	vmopv1common "github.com/vmware-tanzu/vm-operator/api/v1alpha4/common"
	"reflect"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha4"
	pkgcfg "github.com/vmware-tanzu/vm-operator/pkg/config"
	pkgctx "github.com/vmware-tanzu/vm-operator/pkg/context"
	"github.com/vmware-tanzu/vm-operator/pkg/patch"
	"github.com/vmware-tanzu/vm-operator/pkg/record"
)

const (
	finalizerName = "vmoperator.vmware.com/virtualmachinesnapshot"
)

// AddToManager adds this package's controller to the provided manager.
func AddToManager(ctx *pkgctx.ControllerManagerContext, mgr manager.Manager) error {
	var (
		controlledType     = &vmopv1.VirtualMachineSnapshot{}
		controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()

		controllerNameShort = fmt.Sprintf("%s-controller", strings.ToLower(controlledTypeName))
		controllerNameLong  = fmt.Sprintf("%s/%s/%s", ctx.Namespace, ctx.Name, controllerNameShort)
	)

	r := NewReconciler(
		ctx,
		mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName(controlledTypeName),
		record.New(mgr.GetEventRecorderFor(controllerNameLong)),
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(controlledType).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

func NewReconciler(
	ctx context.Context,
	client client.Client,
	logger logr.Logger,
	recorder record.Recorder) *Reconciler {
	return &Reconciler{
		Context:  ctx,
		Client:   client,
		Logger:   logger,
		Recorder: recorder,
	}
}

// Reconciler reconciles a VirtualMachineSnapShot object.
type Reconciler struct {
	client.Client
	Context  context.Context
	Logger   logr.Logger
	Recorder record.Recorder
}

// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachinesnapshots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachinesnapshots/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachines,verbs=get;list;watch;
// +kubebuilder:rbac:groups=vmoperator.vmware.com,resources=virtualmachines/status,verbs=get;update;patch

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx = pkgcfg.JoinContext(ctx, r.Context)

	vmSnapShot := &vmopv1.VirtualMachineSnapshot{}
	if err := r.Get(ctx, req.NamespacedName, vmSnapShot); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	vmSnapShotCtx := &pkgctx.VirtualMachineSnapshotContext{
		Context:                ctx,
		Logger:                 ctrl.Log.WithName("VirtualMachineSnapShot").WithValues("name", req.Name),
		VirtualMachineSnapshot: vmSnapShot,
	}

	patchHelper, err := patch.NewHelper(vmSnapShot, r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to init patch helper for %s: %w", vmSnapShotCtx, err)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, vmSnapShot); err != nil {
			if reterr == nil {
				reterr = err
			}
			vmSnapShotCtx.Logger.Error(err, "patch failed")
		}
	}()

	if !vmSnapShot.DeletionTimestamp.IsZero() {
		// Noop.
		return ctrl.Result{}, nil
	}

	if err := r.ReconcileNormal(vmSnapShotCtx); err != nil {
		vmSnapShotCtx.Logger.Error(err, "Failed to reconcile VirtualMachineSnapShot")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// checkIsSourceValid function checks if the source VM is valid. It is invalid if the VM k8s resource
// doesn't exist, or is not in Created phase.
func (r *Reconciler) checkIsSourceValid(ctx *pkgctx.VirtualMachineSnapshotContext) error {
	// conditions.MarkTrue(vmSnapShot, vmopv1.VirtualMachinePublishRequestConditionSourceValid)
	return nil
}

func (r *Reconciler) ReconcileNormal(ctx *pkgctx.VirtualMachineSnapshotContext) error {
	ctx.Logger.Info("Reconciling VirtualMachineSnapshot")
	vmSnapShot := ctx.VirtualMachineSnapshot

	if !controllerutil.ContainsFinalizer(vmSnapShot, finalizerName) {

		// The finalizer must be present before proceeding in order to ensure that the VirtualMachineSnapshot will
		// be cleaned up. Return immediately after here to let the patcher helper update the
		// object, and then we'll proceed on the next reconciliation.
		controllerutil.AddFinalizer(vmSnapShot, finalizerName)
		return nil
	}

	// return early if phase was already set; nothing to do
	if phase := ctx.VirtualMachineSnapshot.Status.Phase; phase != nil {
		return nil
	}

	vm := &vmopv1.VirtualMachine{}
	ctx.Logger.Info("Fetching VM from snapshot obj:", "vmSnapshot obj", vmSnapShot)
	objKey := client.ObjectKey{Name: vmSnapShot.Spec.Source.Name, Namespace: vmSnapShot.Namespace}
	err := r.Get(ctx, objKey, vm)
	if err != nil {
		ctx.Logger.Error(err, "failed to get VirtualMachine", "vm", objKey)
		return err
	}
	ctx.VM = vm

	if vm.Status.UniqueID == "" {
		err = errors.New("VM hasn't been created and has no uniqueID")
		return err
	}

	objRef := &vmopv1common.LocalObjectRef{
		APIVersion: ctx.VirtualMachineSnapshot.APIVersion,
		Kind:       ctx.VirtualMachineSnapshot.Kind,
		Name:       ctx.VirtualMachineSnapshot.Name,
	}

	// vm object already set with snapshot reference
	if reflect.DeepEqual(vm.Spec.CurrentSnapshot, objRef) {
		return nil
	}

	// get VM again to ensure it's up-to-date.
	err = r.Get(ctx, objKey, vm)
	if err != nil {
		ctx.Logger.Error(err, "failed to get VirtualMachine", "vm", objKey)
		return err
	}

	// patch vm resource with the spec.currentSnapshot
	vmPatch := client.MergeFrom(vm.DeepCopy())
	vm.Spec.CurrentSnapshot = objRef
	if err := r.Patch(ctx, vm, vmPatch); err != nil {
		return fmt.Errorf(
			"failed to patch VM resource %s with current snapshot %s: %w", objKey,
			ctx.VirtualMachineSnapshot.Name, err)
	}

	inProgress := vmopv1.VMSnapshotInProgress
	ctx.VirtualMachineSnapshot.Status.Phase = &inProgress

	return nil
}
