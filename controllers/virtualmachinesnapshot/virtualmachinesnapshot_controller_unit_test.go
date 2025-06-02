// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package virtualmachinesnapshot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha4"

	"github.com/vmware-tanzu/vm-operator/controllers/virtualmachinesnapshot"
	"github.com/vmware-tanzu/vm-operator/pkg/constants/testlabels"
	"github.com/vmware-tanzu/vm-operator/test/builder"
)

func unitTests() {
	Describe(
		"Reconcile",
		Label(
			testlabels.Controller,
			testlabels.API,
		),
		unitTestsReconcile,
	)
}

func unitTestsReconcile() {
	var (
		initObjects []client.Object
		ctx         *builder.UnitTestContextForController

		reconciler *virtualmachinesnapshot.Reconciler
		vmSnapshot *vmopv1.VirtualMachineSnapshot
		vm         *vmopv1.VirtualMachine
	)

	BeforeEach(func() {
		initObjects = nil
		vm = &vmopv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dummy-vm",
				Namespace: "test-namespace",
			},
			Spec: vmopv1.VirtualMachineSpec{
				ImageName:  "dummy-image",
				PowerState: vmopv1.VirtualMachinePowerStateOn,
			},
		}

		vmSnapshot = &vmopv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "snap-1",
				Namespace: "test-namespace",
			},
			Spec: vmopv1.VirtualMachineSnapshotSpec{
				VMRef: corev1.TypedLocalObjectReference{
					APIGroup: &[]string{vmopv1.GroupName}[0],
					Kind:     vm.Kind,
					Name:     vm.Name,
				},
			},
		}
	})

	JustBeforeEach(func() {
		ctx = suite.NewUnitTestContextForController(initObjects...)
		reconciler = virtualmachinesnapshot.NewReconciler(
			ctx,
			ctx.Client,
			ctx.Logger,
			ctx.Recorder,
		)
	})

	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
		initObjects = nil
		reconciler = nil
	})

	Context("Reconcile", func() {
		var (
			err error
		)

		const (
			dummyVmUuid = "unique-vm-id"
		)

		BeforeEach(func() {
			err = nil
			initObjects = append(initObjects, vmSnapshot)
		})

		JustBeforeEach(func() {
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: vmSnapshot.Namespace,
					Name:      vmSnapshot.Name,
				}})
		})

		When("vm does not exist", func() {
			It("returns failure", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		When("vm resource exists but not ready", func() {
			BeforeEach(func() {
				initObjects = append(initObjects, vm)
			})

			It("returns failure", func() {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("VM hasn't been created and has no uniqueID"))
			})
		})

		When("vm ready with different/empty current snapshot ", func() {
			BeforeEach(func() {
				vm.Status.UniqueID = dummyVmUuid
				initObjects = append(initObjects, vm)
			})

			It("returns success", func() {
				Expect(err).ToNot(HaveOccurred())
				objKey := types.NamespacedName{Name: vm.Name, Namespace: vm.Namespace}
				vmObj := &vmopv1.VirtualMachine{}
				Expect(ctx.Client.Get(ctx, objKey, vmObj)).To(Succeed())

				Expect(vmObj.Spec.CurrentSnapshot).To(Equal(&corev1.LocalObjectReference{
					Name: vmSnapshot.Name,
				}))

				snapShotObj := &vmopv1.VirtualMachineSnapshot{}
				snapObjKey := types.NamespacedName{Name: vmSnapshot.Name, Namespace: vmSnapshot.Namespace}
				Expect(ctx.Client.Get(ctx, snapObjKey, snapShotObj)).To(Succeed())

				Expect(snapShotObj.Status.Phase).ToNot(BeNil())
				Expect(*snapShotObj.Status.Phase).To(Equal(vmopv1.VMSnapshotInProgress))
			})
		})

		When("vm ready with matching current snapshot name", func() {
			BeforeEach(func() {
				vm.Status.UniqueID = dummyVmUuid
				vm.Spec.CurrentSnapshot = &corev1.LocalObjectReference{
					Name: vmSnapshot.Name,
				}
				initObjects = append(initObjects, vm)
			})

			It("returns success", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

}
