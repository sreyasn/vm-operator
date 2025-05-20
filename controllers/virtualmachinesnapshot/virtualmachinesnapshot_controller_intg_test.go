// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package virtualmachinesnapshot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha4"
	"github.com/vmware-tanzu/vm-operator/pkg/constants/testlabels"
	"github.com/vmware-tanzu/vm-operator/test/builder"
)

func intgTests() {
	Describe(
		"Reconcile",
		Label(
			testlabels.Controller,
			testlabels.EnvTest,
			testlabels.API,
		),
		intgTestsReconcile,
	)
}

func intgTestsReconcile() {
	var (
		ctx        *builder.IntegrationTestContext
		vmSnapShot *vmopv1.VirtualMachineSnapshot
	)

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()

		vmSnapShot = &vmopv1.VirtualMachineSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "small",
				Namespace: "default",
			},
			Spec: vmopv1.VirtualMachineSnapshotSpec{},
		}
	})

	AfterEach(func() {
		ctx.AfterEach()
	})

	Context("Reconcile", func() {
		BeforeEach(func() {
			Expect(ctx.Client.Create(ctx, vmSnapShot)).To(Succeed())
		})

		AfterEach(func() {
			err := ctx.Client.Delete(ctx, vmSnapShot)
			Expect(err == nil || apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("noop", func() {
		})
	})
}
