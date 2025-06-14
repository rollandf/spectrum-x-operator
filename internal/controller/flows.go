/*
Copyright 2025, NVIDIA CORPORATION & AFFILIATES

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controller

import (
	"fmt"

	"github.com/Mellanox/spectrum-x-operator/pkg/config"
	"github.com/Mellanox/spectrum-x-operator/pkg/exec"
	libnetlink "github.com/Mellanox/spectrum-x-operator/pkg/lib/netlink"
)

const (
	railPeerIP = "rail_peer_ip"
	railUplink = "rail_uplink"
)

//go:generate ../../bin/mockgen -destination mock_flows.go -source flows.go -package controller

type FlowsAPI interface {
	DeleteBridgeDefaultFlows(bridge string) error
	AddHostRailFlows(bridge string, pf string, rail config.HostRail, infraRailSubnet string) error
	AddPodRailFlows(cookie uint64, vf, bridge, podIP, podMAC string) error
	DeletePodRailFlows(cookie uint64, bridge string) error
}

var _ FlowsAPI = &Flows{}

type Flows struct {
	Exec       exec.API
	NetlinkLib libnetlink.NetlinkLib
}

func (f *Flows) DeleteBridgeDefaultFlows(bridge string) error {
	// delete normal action flows - creating a secured bridge is not supported with sriov-network-operator
	// the error is ignored because in non secured bridge there are flows that cannot be deleted, specfically
	// for cookie=0, the first time will work but second reconcile will fail.
	// those are the default flows:
	// 	cookie=0x0, duration=1297.663s, table=254, n_packets=0, n_bytes=0, priority=0,reg0=0x1 actions=controller(reason=)
	// 	cookie=0x0, duration=1297.663s, table=254, n_packets=0, n_bytes=0, priority=2,recirc_id=0 actions=drop
	// 	cookie=0x0, duration=1297.663s, table=254, n_packets=0, n_bytes=0, priority=0,reg0=0x3 actions=drop
	// 	cookie=0x0, duration=1297.663s, table=254, n_packets=0, n_bytes=0, priority=0,reg0=0x2 actions=drop
	// once the bridge is created as secured this code will be removed
	_, _ = f.Exec.Execute(fmt.Sprintf("ovs-ofctl del-flows %s cookie=0x0/-1", bridge))

	return nil
}

func (f *Flows) AddHostRailFlows(bridge string, pf string, rail config.HostRail, infraRailSubnet string) error {
	link, err := f.NetlinkLib.LinkByName(bridge)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %w", bridge, err)
	}
	addrs, err := f.NetlinkLib.IPv4Addresses(link)
	if err != nil {
		return fmt.Errorf("failed to get addresses for interface %s: %w", bridge, err)
	}

	// if we don't have all the addresses assigned we will have partial flows
	// checking for 2 addresses because we don't have a proper api between the controllers
	if len(addrs) != 2 {
		return fmt.Errorf("expected 2 addresses for interface %s, got %s", bridge, addrs)
	}

	for _, addr := range addrs {
		flow := fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,arp,arp_tpa=%s,actions=output:local"`,
			bridge, defaultPriority, hostConfigCookie, addr.IP)
		if _, err := f.Exec.Execute(flow); err != nil {
			return fmt.Errorf("failed to exec [%s]: %s", flow, err)
		}

		flow = fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,ip,nw_dst=%s,actions=output:local"`,
			bridge, defaultPriority, hostConfigCookie, addr.IP)
		if _, err := f.Exec.Execute(flow); err != nil {
			return fmt.Errorf("failed to exec [%s]: %s", flow, err)
		}
	}

	// TOR is the gateway for the outer ip
	flow := fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,arp,arp_tpa=%s,actions=output:%s"`,
		bridge, defaultPriority, hostConfigCookie, rail.PeerLeafPortIP, pf)
	if _, err := f.Exec.Execute(flow); err != nil {
		return fmt.Errorf("failed to exec [%s]: %s", flow, err)
	}

	flow = fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,`+
		`ip,in_port=local,nw_dst=%s,actions=output:%s"`,
		bridge, defaultPriority, hostConfigCookie, infraRailSubnet, pf)
	if _, err := f.Exec.Execute(flow); err != nil {
		return fmt.Errorf("failed to exec [%s]: %s", flow, err)
	}

	return nil
}

func (f *Flows) AddPodRailFlows(cookie uint64, vf, bridge, podIP, podMAC string) error {
	// ovs-ofctl add-flow -OOpenFlow13 $RAIL_BR "table=0, arp,arp_tpa=${CONTAINER_IP} actions=output:${REP_PORT}"
	flow := fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,arp,arp_tpa=%s,actions=output:%s"`,
		bridge, defaultPriority, cookie, podIP, vf)
	if _, err := f.Exec.Execute(flow); err != nil {
		return fmt.Errorf("failed to add flows to bridge [%s]: %v", bridge, err)
	}

	link, err := f.NetlinkLib.LinkByName(bridge)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %w", bridge, err)
	}

	bridgeMAC := link.Attrs().HardwareAddr

	torIP, err := f.Exec.Execute(fmt.Sprintf("ovs-vsctl br-get-external-id %s %s", bridge, railPeerIP))
	if err != nil {
		return fmt.Errorf("failed to get tor ip for bridge [%s]: %v", bridge, err)
	}

	if torIP == "" {
		return fmt.Errorf("tor ip is empty for bridge [%s], set [%s] external_id on the bridge", bridge, railPeerIP)
	}

	torMAC, err := f.getTorMac(torIP)
	if err != nil {
		return fmt.Errorf("failed to get tor mac for bridge [%s]: %v", bridge, err)
	}

	uplink, err := f.Exec.Execute(fmt.Sprintf("ovs-vsctl br-get-external-id %s %s", bridge, railUplink))
	if err != nil {
		return fmt.Errorf("failed to get rail uplink for bridge [%s]: %v", bridge, err)
	}

	if uplink == "" {
		return fmt.Errorf("uplink is empty for bridge [%s], set [%s] external_id on the bridge", bridge, railUplink)
	}

	// ovs-ofctl add-flow -OOpenFlow13 $RAIL_BR "table=0,ip,in_port=${REP_PORT},
	// actions=mod_dl_src=${ROUTER_MAC}, mod_dl_dst=${ROUTER_NH_MAC},dec_ttl, output=${PF_PORT}"
	// setting the priority to avoid conflicts with a more specific flows
	flow = fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,ip,in_port=%s,`+
		`actions=mod_dl_src=%s,mod_dl_dst=%s,dec_ttl,output:%s"`,
		bridge, defaultPriority/2, cookie, vf, bridgeMAC, torMAC, uplink)
	if _, err := f.Exec.Execute(flow); err != nil {
		return fmt.Errorf("failed to add flows to bridge [%s] flow [%s] %v", bridge, flow, err)
	}

	// ovs-ofctl add-flow -OOpenFlow13 $RAIL_BR "table=0,ip,nw_dst=${CONTAINER_IP},
	// actions=mod_dl_src=${ROUTER_MAC},mod_dl_dst=${CONTAINER_MAC},dec_ttl, output=${REP_PORT}"
	flow = fmt.Sprintf(`ovs-ofctl add-flow %s "table=0,priority=%d,cookie=0x%x,ip,nw_dst=%s,`+
		`actions=mod_dl_src=%s,mod_dl_dst=%s,dec_ttl,output:%s"`,
		bridge, defaultPriority, cookie, podIP, bridgeMAC, podMAC, vf)
	if _, err := f.Exec.Execute(flow); err != nil {
		return fmt.Errorf("failed to add flows to bridge [%s] flow [%s]: %v", bridge, flow, err)
	}

	return nil
}

func (f *Flows) DeletePodRailFlows(cookie uint64, bridge string) error {
	flow := fmt.Sprintf(`ovs-ofctl del-flows %s cookie=0x%x/-1`, bridge, cookie)
	_, err := f.Exec.Execute(flow)
	return err
}

func (f *Flows) getTorMac(torIP string) (string, error) {
	// nsenter --target 1 --net -- arping 2.0.0.3 -c 1
	// nsenter --target 1 --net -- ip neighbor | grep 2.0.0.3 | awk '{print $5}'
	// TODO: check why it always return an error
	reply, _ := f.Exec.ExecutePrivileged(fmt.Sprintf(`arping %s -c 1 | grep "reply from" | awk '{print $5}' | tr -d '[]'`,
		torIP))
	// if err != nil {
	// 	logr.Error(err, fmt.Sprintf("failed to exec: arping %s -c 1", rail.Tor))
	// 	return "", err
	// }

	if reply == "" {
		return "", fmt.Errorf("no reply from arping %s", torIP)
	}

	return reply, nil
}
