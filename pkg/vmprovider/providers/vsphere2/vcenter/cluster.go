// Copyright (c) 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vcenter

import (
	goctx "context"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
)

// ClusterMinCPUFreq returns the minimum frequency across all the hosts in the cluster. This is needed to
// convert the CPU requirements specified in cores to MHz. vSphere core is assumed to be equivalent to the
// value of min frequency. This function is adapted from wcp schedext.
func ClusterMinCPUFreq(ctx goctx.Context, cluster *object.ClusterComputeResource) (uint64, error) {
	var cr mo.ComputeResource
	if err := cluster.Properties(ctx, cluster.Reference(), []string{"host"}, &cr); err != nil {
		return 0, err
	}

	if len(cr.Host) == 0 {
		return 0, nil
	}

	var hosts []mo.HostSystem
	pc := property.DefaultCollector(cluster.Client())
	if err := pc.Retrieve(ctx, cr.Host, []string{"summary"}, &hosts); err != nil {
		return 0, err
	}

	var minFreq uint64
	for _, h := range hosts {
		if hw := h.Summary.Hardware; hw != nil {
			hostCPUMHz := uint64(hw.CpuMhz)
			if hostCPUMHz < minFreq || minFreq == 0 {
				minFreq = hostCPUMHz
			}
		}
	}

	return minFreq, nil
}
