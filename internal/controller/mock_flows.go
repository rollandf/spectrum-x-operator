// Code generated by MockGen. DO NOT EDIT.
// Source: flows.go

// Package controller is a generated GoMock package.
package controller

import (
	reflect "reflect"

	config "github.com/Mellanox/spectrum-x-operator/pkg/config"
	gomock "github.com/golang/mock/gomock"
	v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

// MockFlowsAPI is a mock of FlowsAPI interface.
type MockFlowsAPI struct {
	ctrl     *gomock.Controller
	recorder *MockFlowsAPIMockRecorder
}

// MockFlowsAPIMockRecorder is the mock recorder for MockFlowsAPI.
type MockFlowsAPIMockRecorder struct {
	mock *MockFlowsAPI
}

// NewMockFlowsAPI creates a new mock instance.
func NewMockFlowsAPI(ctrl *gomock.Controller) *MockFlowsAPI {
	mock := &MockFlowsAPI{ctrl: ctrl}
	mock.recorder = &MockFlowsAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFlowsAPI) EXPECT() *MockFlowsAPIMockRecorder {
	return m.recorder
}

// AddHostRailFlows mocks base method.
func (m *MockFlowsAPI) AddHostRailFlows(bridge, pf string, rail config.HostRail) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddHostRailFlows", bridge, pf, rail)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddHostRailFlows indicates an expected call of AddHostRailFlows.
func (mr *MockFlowsAPIMockRecorder) AddHostRailFlows(bridge, pf, rail interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddHostRailFlows", reflect.TypeOf((*MockFlowsAPI)(nil).AddHostRailFlows), bridge, pf, rail)
}

// AddPodRailFlows mocks base method.
func (m *MockFlowsAPI) AddPodRailFlows(rail *config.HostRail, cfg *config.Config, ns *v1.NetworkStatus, bridge, iface string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddPodRailFlows", rail, cfg, ns, bridge, iface)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddPodRailFlows indicates an expected call of AddPodRailFlows.
func (mr *MockFlowsAPIMockRecorder) AddPodRailFlows(rail, cfg, ns, bridge, iface interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddPodRailFlows", reflect.TypeOf((*MockFlowsAPI)(nil).AddPodRailFlows), rail, cfg, ns, bridge, iface)
}

// DeleteBridgeDefaultFlows mocks base method.
func (m *MockFlowsAPI) DeleteBridgeDefaultFlows(bridge string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteBridgeDefaultFlows", bridge)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteBridgeDefaultFlows indicates an expected call of DeleteBridgeDefaultFlows.
func (mr *MockFlowsAPIMockRecorder) DeleteBridgeDefaultFlows(bridge interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteBridgeDefaultFlows", reflect.TypeOf((*MockFlowsAPI)(nil).DeleteBridgeDefaultFlows), bridge)
}
