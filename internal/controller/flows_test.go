// Copyright 2025 NVIDIA CORPORATION & AFFILIATES
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"net"
	"strings"

	"github.com/Mellanox/spectrum-x-operator/pkg/exec"
	mock_netlink "github.com/Mellanox/spectrum-x-operator/pkg/lib/netlink/mocks"
	gomock "github.com/golang/mock/gomock"
	vishvananda_netlink "github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type containsSubstringMatcher struct {
	substring string
}

func matchSubstring(substring string) gomock.Matcher {
	return &containsSubstringMatcher{substring}
}

func (m *containsSubstringMatcher) Matches(x interface{}) bool {
	s, ok := x.(string)
	if !ok {
		return false
	}
	return strings.Contains(s, m.substring)
}

func (m *containsSubstringMatcher) String() string {
	return "contains substring " + m.substring
}

var _ = Describe("Flows", func() {
	var (
		flows       *Flows
		execMock    *exec.MockAPI
		netlinkMock *mock_netlink.MockNetlinkLib
		ctrl        *gomock.Controller
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		execMock = exec.NewMockAPI(ctrl)
		netlinkMock = mock_netlink.NewMockNetlinkLib(ctrl)
		flows = &Flows{
			Exec:       execMock,
			NetlinkLib: netlinkMock,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("DeleteBridgeDefaultFlows", func() {
		It("should delete flows with cookie=0", func() {
			execMock.EXPECT().Execute("ovs-ofctl del-flows br-rail1 cookie=0x0/-1").Return("", nil)
			err := flows.DeleteBridgeDefaultFlows("br-rail1")
			Expect(err).Should(Succeed())
		})

		It("should not return error if ovs-ofctl fails", func() {
			execMock.EXPECT().Execute("ovs-ofctl del-flows br-rail1 cookie=0x0/-1").Return("", fmt.Errorf("failed to delete flows"))
			err := flows.DeleteBridgeDefaultFlows("br-rail1")
			Expect(err).Should(Succeed())
		})
	})

	Context("AddPodRailFlows", func() {
		var (
			mockLink *mock_netlink.MockLink
		)

		BeforeEach(func() {
			mockLink = mock_netlink.NewMockLink(ctrl)
			linkAttrs := &vishvananda_netlink.LinkAttrs{
				Name:         "br-rail1",
				HardwareAddr: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			}
			mockLink.EXPECT().Attrs().Return(linkAttrs).AnyTimes()
		})

		It("should add flows for pod rail", func() {
			// Mock getting link
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)

			// Mock getting TOR IP
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("192.168.1.1", nil)

			// Mock getting TOR MAC
			execMock.EXPECT().ExecutePrivileged(gomock.Any()).Return("00:11:22:33:44:55", nil)

			// Mock getting uplink
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_uplink").Return("p0", nil)

			// Mock adding flows
			execMock.EXPECT().Execute(gomock.Any()).Return("", nil).Times(3)

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(Succeed())
		})

		It("should return error if fails to add arp flow", func() {
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", fmt.Errorf("failed to add arp flow"))
			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to add flows to bridge"))
		})

		It("should return error if fails to add ip flow", func() {
			// First ARP flow succeeds
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("192.168.1.1", nil)
			execMock.EXPECT().ExecutePrivileged(gomock.Any()).Return("00:11:22:33:44:55", nil)
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_uplink").Return("p0", nil)
			// Second IP flow fails
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", fmt.Errorf("failed to add ip flow"))

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to add flows to bridge"))
		})

		It("should return error if fails to add pod flow", func() {
			// First ARP flow succeeds
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("192.168.1.1", nil)
			execMock.EXPECT().ExecutePrivileged(gomock.Any()).Return("00:11:22:33:44:55", nil)
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_uplink").Return("p0", nil)
			// Second IP flow succeeds
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			// Third pod flow fails
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", fmt.Errorf("failed to add pod flow"))

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to add flows to bridge"))
		})

		It("should return error if fails to get link", func() {
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(nil, fmt.Errorf("failed to get link"))
			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to get interface"))
		})

		It("should return error if failed to get external ids", func() {
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			execMock.EXPECT().Execute(matchSubstring("ovs-vsctl br-get-external-id ")).
				Return("", fmt.Errorf("failed to get external id"))

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to get tor ip for bridge"))
		})

		It("should return error if TOR IP not found", func() {
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			// First call is to add the ARP flow
			execMock.EXPECT().Execute(`ovs-ofctl add-flow br-rail1 "table=0,priority=32768,cookie=0x5,arp,arp_tpa=10.0.0.1,actions=output:vf0"`).Return("", nil)
			// Then it checks for TOR IP
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("", nil)

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("tor ip is empty"))
		})

		It("should return error if uplink not found", func() {
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			// First call is to add the ARP flow
			execMock.EXPECT().Execute(`ovs-ofctl add-flow br-rail1 "table=0,priority=32768,cookie=0x5,arp,arp_tpa=10.0.0.1,actions=output:vf0"`).Return("", nil)
			// Then it checks for TOR IP
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("192.168.1.1", nil)
			// Then it gets TOR MAC
			execMock.EXPECT().ExecutePrivileged(gomock.Any()).Return("00:11:22:33:44:55", nil)
			// Finally it checks for uplink
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_uplink").Return("", nil)

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("uplink is empty"))
		})

		It("should return error if fails to get TOR MAC", func() {
			// First ARP flow succeeds
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("192.168.1.1", nil)
			// TOR MAC retrieval fails
			execMock.EXPECT().ExecutePrivileged(gomock.Any()).Return("", fmt.Errorf("failed to get TOR MAC"))

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to get tor mac for bridge"))
		})

		It("should return error if fails to get uplink", func() {
			// First ARP flow succeeds
			execMock.EXPECT().Execute(matchSubstring("ovs-ofctl add-flow br-rail1")).Return("", nil)
			netlinkMock.EXPECT().LinkByName("br-rail1").Return(mockLink, nil)
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_peer_ip").Return("192.168.1.1", nil)
			// TOR MAC retrieval succeeds
			execMock.EXPECT().ExecutePrivileged(gomock.Any()).Return("00:11:22:33:44:55", nil)
			// Uplink retrieval fails
			execMock.EXPECT().Execute("ovs-vsctl br-get-external-id br-rail1 rail_uplink").Return("", fmt.Errorf("failed to get uplink"))

			err := flows.AddPodRailFlows(0x5, "vf0", "br-rail1", "10.0.0.1", "00:11:22:33:44:66")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("failed to get rail uplink for bridge"))
		})
	})

	Context("DeletePodRailFlows", func() {
		It("should delete flows on pod deletion", func() {
			execMock.EXPECT().Execute("ovs-ofctl del-flows br-rail1 cookie=0x5/-1").Return("", nil)
			err := flows.DeletePodRailFlows(0x5, "br-rail1")
			Expect(err).Should(Succeed())
		})

		It("should return error if ovs-ofctl fails", func() {
			execMock.EXPECT().Execute("ovs-ofctl del-flows br-rail1 cookie=0x5/-1").Return("", fmt.Errorf("failed to delete flows"))
			err := flows.DeletePodRailFlows(0x5, "br-rail1")
			Expect(err).Should(HaveOccurred())
		})
	})
})
