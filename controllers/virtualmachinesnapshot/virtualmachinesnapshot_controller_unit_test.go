// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package virtualmachinesnapshot_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
	)

	BeforeEach(func() {
		initObjects = nil
		vmSnapshot = &vmopv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dummy-vmsnapshot",
			},
		}
	})

	JustBeforeEach(func() {
		initObjects = append(initObjects, vmSnapshot)
		ctx = suite.NewUnitTestContextForController(initObjects...)
		reconciler = virtualmachinesnapshot.NewReconciler(
			ctx,
			ctx.Client,
			ctx.Logger,
			ctx.Recorder,
		)
	})

	Context("Reconcile", func() {
		var (
			err  error
			name string
		)

		BeforeEach(func() {
			err = nil
			name = vmSnapshot.Name
		})

		JustBeforeEach(func() {
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: vmSnapshot.Namespace,
					Name:      name,
				}})
		})

		When("Deleted", func() {
			BeforeEach(func() {
				vmSnapshot.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				vmSnapshot.Finalizers = append(vmSnapshot.Finalizers, "fake.com/finalizer")
			})
			It("returns success", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("Normal", func() {
			It("returns success", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("vmSnapshot not found", func() {
			BeforeEach(func() {
				name = "invalid"
			})
			It("ignores the error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

}
