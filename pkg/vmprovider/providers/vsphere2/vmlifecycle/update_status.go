// Copyright (c) 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vmlifecycle

import (
	goctx "context"
	"fmt"
	"net"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/util/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	"github.com/vmware-tanzu/vm-operator/api/v1alpha2/common"
	conditions "github.com/vmware-tanzu/vm-operator/pkg/conditions2"
	"github.com/vmware-tanzu/vm-operator/pkg/context"
	"github.com/vmware-tanzu/vm-operator/pkg/lib"
	"github.com/vmware-tanzu/vm-operator/pkg/topology"
	"github.com/vmware-tanzu/vm-operator/pkg/vmprovider/providers/vsphere2/virtualmachine"
)

var (
	// The minimum properties needed to be retrieved in order to populate the Status. Callers may
	// provide a MO with more. This often saves us a second round trip in the common steady state.
	vmStatusPropertiesSelector = []string{"config.changeTrackingEnabled", "guest", "summary"}
)

func UpdateStatus(
	vmCtx context.VirtualMachineContextA2,
	k8sClient ctrlclient.Client,
	vcVM *object.VirtualMachine,
	vmMO *mo.VirtualMachine) error {

	// This is implicitly true: ensure the condition is set since it is how we determine the old v1a1 Phase.
	conditions.MarkTrue(vmCtx.VM, vmopv1.VirtualMachineConditionCreated)
	// TODO: Might set other "prereq" conditions too for version conversion but we'd have to fib a little.

	if vmMO == nil {
		// In the common case, our caller will have already gotten the MO properties in order to determine
		// if it had any reconciliation to do, and there was nothing to do since the VM is in the steady
		// state so that MO is still entirely valid here.
		// NOTE: The properties must have been retrieved with at least vmStatusPropertiesSelector.
		vmMO = &mo.VirtualMachine{}
		if err := vcVM.Properties(vmCtx, vcVM.Reference(), vmStatusPropertiesSelector, vmMO); err != nil {
			// Leave the current Status unchanged for now.
			return fmt.Errorf("failed to get VM properties for status update: %w", err)
		}
	}

	var errs []error
	var err error
	vm := vmCtx.VM
	summary := vmMO.Summary

	vm.Status.PowerState = convertPowerState(summary.Runtime.PowerState)
	vm.Status.UniqueID = vcVM.Reference().Value
	vm.Status.BiosUUID = summary.Config.Uuid
	vm.Status.InstanceUUID = summary.Config.InstanceUuid
	vm.Status.Network = getGuestNetworkStatus(vmMO.Guest)

	vm.Status.Host, err = getRuntimeHostHostname(vmCtx, vcVM, summary.Runtime.Host)
	if err != nil {
		errs = append(errs, err)
	}

	MarkVMToolsRunningStatusCondition(vmCtx.VM, vmMO.Guest)
	MarkCustomizationInfoCondition(vmCtx.VM, vmMO.Guest)

	if config := vmMO.Config; config != nil {
		vm.Status.ChangeBlockTracking = config.ChangeTrackingEnabled
	} else {
		vm.Status.ChangeBlockTracking = nil
	}

	if lib.IsWcpFaultDomainsFSSEnabled() {
		zoneName := vm.Labels[topology.KubernetesTopologyZoneLabelKey]
		if zoneName == "" {
			cluster, err := virtualmachine.GetVMClusterComputeResource(vmCtx, vcVM)
			if err != nil {
				errs = append(errs, err)
			} else {
				zoneName, err = topology.LookupZoneForClusterMoID(vmCtx, k8sClient, cluster.Reference().Value)
				if err != nil {
					errs = append(errs, err)
				} else {
					if vm.Labels == nil {
						vm.Labels = map[string]string{}
					}
					vm.Labels[topology.KubernetesTopologyZoneLabelKey] = zoneName
				}
			}
		}

		if zoneName != "" {
			vm.Status.Zone = zoneName
		}
	}

	return k8serrors.NewAggregate(errs)
}

func getRuntimeHostHostname(
	ctx goctx.Context,
	vcVM *object.VirtualMachine,
	host *types.ManagedObjectReference) (string, error) {

	if host != nil {
		return object.NewHostSystem(vcVM.Client(), *host).ObjectName(ctx)
	}
	return "", nil
}

func getGuestNetworkStatus(guestInfo *types.GuestInfo) *vmopv1.VirtualMachineNetworkStatus {
	if guestInfo == nil {
		return nil
	}

	status := &vmopv1.VirtualMachineNetworkStatus{}

	if ipAddr := guestInfo.IpAddress; ipAddr != "" {
		// TODO: Filter out local addresses.
		if net.ParseIP(ipAddr).To4() != nil {
			status.PrimaryIP4 = ipAddr
		} else {
			status.PrimaryIP6 = ipAddr
		}
	}

	if len(guestInfo.Net) > 0 {
		status.Interfaces = make([]vmopv1.VirtualMachineNetworkInterfaceStatus, 0, len(guestInfo.Net))
		for i := range guestInfo.Net {
			status.Interfaces = append(status.Interfaces, guestNicInfoToInterfaceStatus(i, &guestInfo.Net[i]))
		}
	}

	if len(guestInfo.IpStack) > 0 {
		status.VirtualMachineNetworkIPStackStatus = guestIPStackInfoToIPStackStatus(&guestInfo.IpStack[0])
	}

	return status
}

func guestNicInfoToInterfaceStatus(idx int, guestNicInfo *types.GuestNicInfo) vmopv1.VirtualMachineNetworkInterfaceStatus {
	status := vmopv1.VirtualMachineNetworkInterfaceStatus{}

	// TODO: What name exactly? The CRD name may be the most useful here but hard to line that up.
	// BMV: DeviceConfigId will be -1 for our pseudo-y interfaces. Most likely want to just skip those devices.
	status.Name = fmt.Sprintf("nic-%d-%d", idx, guestNicInfo.DeviceConfigId)
	status.IP.MACAddr = guestNicInfo.MacAddress

	if guestIPConfig := guestNicInfo.IpConfig; guestIPConfig != nil {
		ip := &status.IP

		ip.AutoConfigurationEnabled = guestIPConfig.AutoConfigurationEnabled
		ip.Addresses = convertNetIPConfigInfoIPAddresses(guestIPConfig.IpAddress)

		if guestIPConfig.Dhcp != nil {
			ip.DHCP = convertNetDhcpConfigInfo(guestIPConfig.Dhcp)
		}
	}

	if dnsConfig := guestNicInfo.DnsConfig; dnsConfig != nil {
		status.DNS = convertNetDNSConfigInfo(dnsConfig)
	}

	return status
}

func guestIPStackInfoToIPStackStatus(guestIPStack *types.GuestStackInfo) vmopv1.VirtualMachineNetworkIPStackStatus {
	status := vmopv1.VirtualMachineNetworkIPStackStatus{}

	if dhcpConfig := guestIPStack.DhcpConfig; dhcpConfig != nil {
		status.DHCP = convertNetDhcpConfigInfo(dhcpConfig)
	}

	if dnsConfig := guestIPStack.DnsConfig; dnsConfig != nil {
		status.DNS = convertNetDNSConfigInfo(dnsConfig)
	}

	if ipRouteConfig := guestIPStack.IpRouteConfig; ipRouteConfig != nil {
		status.IPRoutes = convertNetIPRouteConfigInfo(ipRouteConfig)
	}

	status.KernelConfig = convertKeyValueSlice(guestIPStack.IpStackConfig)

	return status
}

func convertPowerState(powerState types.VirtualMachinePowerState) vmopv1.VirtualMachinePowerState {
	switch powerState {
	case types.VirtualMachinePowerStatePoweredOff:
		return vmopv1.VirtualMachinePowerStateOff
	case types.VirtualMachinePowerStatePoweredOn:
		return vmopv1.VirtualMachinePowerStateOn
	case types.VirtualMachinePowerStateSuspended:
		return vmopv1.VirtualMachinePowerStateSuspended
	}
	return ""
}

func convertNetIPConfigInfoIPAddresses(ipAddresses []types.NetIpConfigInfoIpAddress) []vmopv1.VirtualMachineNetworkInterfaceIPAddrStatus {
	if len(ipAddresses) == 0 {
		return nil
	}

	out := make([]vmopv1.VirtualMachineNetworkInterfaceIPAddrStatus, 0, len(ipAddresses))
	for _, guestIPAddr := range ipAddresses {
		ipAddrStatus := vmopv1.VirtualMachineNetworkInterfaceIPAddrStatus{
			Address: guestIPAddr.IpAddress,
			Origin:  guestIPAddr.Origin,
			State:   guestIPAddr.State,
		}
		if guestIPAddr.Lifetime != nil {
			ipAddrStatus.Lifetime = metav1.NewTime(*guestIPAddr.Lifetime)
		}

		out = append(out, ipAddrStatus)
	}
	return out
}

func convertNetDNSConfigInfo(dnsConfig *types.NetDnsConfigInfo) vmopv1.VirtualMachineNetworkDNSStatus {
	return vmopv1.VirtualMachineNetworkDNSStatus{
		DHCP:          dnsConfig.Dhcp,
		DomainName:    dnsConfig.DomainName,
		HostName:      dnsConfig.HostName,
		Nameservers:   dnsConfig.IpAddress,
		SearchDomains: dnsConfig.SearchDomain,
	}
}

func convertNetDhcpConfigInfo(dhcpConfig *types.NetDhcpConfigInfo) vmopv1.VirtualMachineNetworkDHCPStatus {
	status := vmopv1.VirtualMachineNetworkDHCPStatus{}

	if ipv4 := dhcpConfig.Ipv4; ipv4 != nil {
		status.IP4.Enabled = ipv4.Enable
		status.IP4.Config = convertKeyValueSlice(ipv4.Config)
	}

	if ipv6 := dhcpConfig.Ipv6; ipv6 != nil {
		status.IP6.Enabled = ipv6.Enable
		status.IP6.Config = convertKeyValueSlice(ipv6.Config)
	}

	return status
}

func convertNetIPRouteConfigInfo(routeConfig *types.NetIpRouteConfigInfo) []vmopv1.VirtualMachineNetworkIPRouteStatus {
	if len(routeConfig.IpRoute) == 0 {
		return nil
	}

	out := make([]vmopv1.VirtualMachineNetworkIPRouteStatus, 0, len(routeConfig.IpRoute))
	for _, ipRoute := range routeConfig.IpRoute {
		out = append(out, vmopv1.VirtualMachineNetworkIPRouteStatus{
			Gateway: vmopv1.VirtualMachineNetworkIPRouteGatewayStatus{
				Device:  ipRoute.Gateway.Device,
				Address: ipRoute.Gateway.IpAddress,
			},
			NetworkAddress: fmt.Sprintf("%s/%d", ipRoute.Network, ipRoute.PrefixLength), // TODO: Verify we don't have to mask Network
		})
	}
	return out
}

func convertKeyValueSlice(s []types.KeyValue) []common.KeyValuePair {
	if len(s) == 0 {
		return nil
	}

	out := make([]common.KeyValuePair, 0, len(s))
	for i := range s {
		out = append(out, common.KeyValuePair{Key: s[i].Key, Value: s[i].Value})
	}
	return out
}

func MarkVMToolsRunningStatusCondition(
	vm *vmopv1.VirtualMachine,
	guestInfo *types.GuestInfo) {

	if guestInfo == nil || guestInfo.ToolsRunningStatus == "" {
		conditions.MarkUnknown(vm, vmopv1.VirtualMachineToolsCondition, "NoGuestInfo", "")
		return
	}

	switch guestInfo.ToolsRunningStatus {
	case string(types.VirtualMachineToolsRunningStatusGuestToolsNotRunning):
		msg := "VMware Tools is not running"
		conditions.MarkFalse(vm, vmopv1.VirtualMachineToolsCondition, vmopv1.VirtualMachineToolsNotRunningReason, msg)
	case string(types.VirtualMachineToolsRunningStatusGuestToolsRunning), string(types.VirtualMachineToolsRunningStatusGuestToolsExecutingScripts):
		conditions.MarkTrue(vm, vmopv1.VirtualMachineToolsCondition)
	default:
		msg := "Unexpected VMware Tools running status"
		conditions.MarkUnknown(vm, vmopv1.VirtualMachineToolsCondition, "Unknown", msg)
	}
}

func MarkCustomizationInfoCondition(vm *vmopv1.VirtualMachine, guestInfo *types.GuestInfo) {
	if guestInfo == nil || guestInfo.CustomizationInfo == nil {
		conditions.MarkUnknown(vm, vmopv1.GuestCustomizationCondition, "NoGuestInfo", "")
		return
	}

	switch guestInfo.CustomizationInfo.CustomizationStatus {
	case string(types.GuestInfoCustomizationStatusTOOLSDEPLOYPKG_IDLE), "":
		conditions.MarkTrue(vm, vmopv1.GuestCustomizationCondition)
	case string(types.GuestInfoCustomizationStatusTOOLSDEPLOYPKG_PENDING):
		conditions.MarkFalse(vm, vmopv1.GuestCustomizationCondition, vmopv1.GuestCustomizationPendingReason, "")
	case string(types.GuestInfoCustomizationStatusTOOLSDEPLOYPKG_RUNNING):
		conditions.MarkFalse(vm, vmopv1.GuestCustomizationCondition, vmopv1.GuestCustomizationRunningReason, "")
	case string(types.GuestInfoCustomizationStatusTOOLSDEPLOYPKG_SUCCEEDED):
		conditions.MarkTrue(vm, vmopv1.GuestCustomizationCondition)
	case string(types.GuestInfoCustomizationStatusTOOLSDEPLOYPKG_FAILED):
		errorMsg := guestInfo.CustomizationInfo.ErrorMsg
		if errorMsg == "" {
			errorMsg = "vSphere VM Customization failed due to an unknown error."
		}
		conditions.MarkFalse(vm, vmopv1.GuestCustomizationCondition, vmopv1.GuestCustomizationFailedReason, errorMsg)
	default:
		errorMsg := guestInfo.CustomizationInfo.ErrorMsg
		if errorMsg == "" {
			errorMsg = "Unexpected VM Customization status"
		}
		conditions.MarkFalse(vm, vmopv1.GuestCustomizationCondition, "Unknown", errorMsg)
	}
}
