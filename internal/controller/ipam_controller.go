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
	"context"
	"fmt"
	"net"
	"time"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Mellanox/spectrum-x-operator/pkg/config"

	nvipamv1 "github.com/Mellanox/nvidia-k8s-ipam/api/v1alpha1"
)

const (
	spectrumXIPAMControllerName = "spectrumxipamcontroller"
	cidrPoolType                = "CIDRPool"
	spectrumXFinalizer          = "spectrumx.nvidia.com/ipam"
)

// NvIPAMReconciler reconciles Spectrum-X-Config from ConfigMap
type NvIPAMReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	ConfigMapNamespace string
	ConfigMapName      string
	CIDRPoolsNamespace string
}

//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps/finalizers,verbs=update
//+kubebuilder:rbac:groups=nv-ipam.nvidia.com,resources=cidrpools,verbs=get;list;patch;create;delete;deletecollection
//+kubebuilder:rbac:groups="",resources=events,verbs=get;create;patch;update

func (r *NvIPAMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	logr.Info("Reconciling Spectrum-X ConfigMap")
	// Fetch the ConfigMap
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, req.NamespacedName, cm)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			// Return early if the object is not found.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// Handle deletion reconciliation loop.
	if !cm.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, cm)
	}

	spectrumXConfig, err := config.ParseConfig(cm.Data[config.ConfigMapKey])
	if err != nil {
		logr.Info("Failed to parse Spectrum-X ConfigMap Data")
		return ctrl.Result{}, err
	}
	logr.Info("Parsed Spectrum-X Config", "config", spectrumXConfig)

	// Add finalizer if not set.
	if !controllerutil.ContainsFinalizer(cm, spectrumXFinalizer) {
		logr.Info("Adding finalizer to configmap")
		controllerutil.AddFinalizer(cm, spectrumXFinalizer)
		cm.ObjectMeta.ManagedFields = nil
		cm.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
		if err := r.Client.Patch(ctx, cm, client.Apply, client.ForceOwnership, client.FieldOwner(spectrumXIPAMControllerName)); err != nil {
			return ctrl.Result{}, fmt.Errorf("error while adding finalizer to configmap %w", err)
		}
		return ctrl.Result{}, nil
	}

	err = r.reconcileCIDRPools(ctx, spectrumXConfig)
	if err != nil {
		logr.Info("Failed to reconcile CIDRPool")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileCIDRPools generates CIDRPools according to defined Rails
func (r *NvIPAMReconciler) reconcileCIDRPools(ctx context.Context, cfg *config.Config) error {
	logr := log.FromContext(ctx)
	for _, rail := range cfg.SpectrumXNetworks.Rails {
		pool, err := r.generateCIDRPool(&rail, cfg.Hosts, cfg.SpectrumXNetworks.CrossRailSubnet)
		if err != nil {
			logr.Error(err, "Fail to build CIDRPool", "rail", rail)
			return err
		}
		logr.Info("Creating CIDRPool", "name", pool.Name)
		if err := r.Client.Patch(ctx, pool, client.Apply, client.ForceOwnership, client.FieldOwner(spectrumXIPAMControllerName)); err != nil {
			return fmt.Errorf("error while patching %s %s: %w", pool.GetObjectKind().GroupVersionKind().String(), client.ObjectKeyFromObject(pool), err)
		}
	}
	return nil
}

// generateCIDRPool generates a CIDRPool object for the given Rail
func (r *NvIPAMReconciler) generateCIDRPool(rail *config.Rail, hosts []config.Host, crossRailNet string) (*nvipamv1.CIDRPool, error) {
	staticAllocations, err := getHostsAllocation(rail.Name, hosts)
	if err != nil {
		return nil, err
	}
	pool := &nvipamv1.CIDRPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rail.Name,
			Namespace: r.CIDRPoolsNamespace,
			Labels: map[string]string{
				spectrumXIPAMControllerName: "",
			},
		},
		Spec: nvipamv1.CIDRPoolSpec{
			CIDR:                 rail.Subnet,
			GatewayIndex:         ptr.To[int32](0),
			PerNodeNetworkPrefix: 31,
			StaticAllocations:    staticAllocations,
			Routes: []nvipamv1.Route{
				{
					Dst: rail.Subnet,
				},
				{
					Dst: crossRailNet,
				},
			},
		},
	}
	pool.ObjectMeta.ManagedFields = nil
	pool.SetGroupVersionKind(nvipamv1.GroupVersion.WithKind(cidrPoolType))
	return pool, nil
}

// getHostsAllocation  computes the static allocations for the hosts for a specific Rail
func getHostsAllocation(rail string, hosts []config.Host) ([]nvipamv1.CIDRPoolStaticAllocation, error) {
	allocs := make([]nvipamv1.CIDRPoolStaticAllocation, 0)
	for _, h := range hosts {
		for _, r := range h.Rails {
			if r.Name == rail {
				_, ipNet, err := net.ParseCIDR(r.Network)
				if err != nil {
					return allocs, err
				}
				gw := nvipamv1.GetGatewayForSubnet(ipNet, 1)
				alloc := nvipamv1.CIDRPoolStaticAllocation{
					NodeName: h.HostID,
					Prefix:   r.Network,
					Gateway:  gw,
				}
				allocs = append(allocs, alloc)
			}
		}
	}
	return allocs, nil
}

// reconcileDelete delete created CIDRPools and remove finalizer
func (r *NvIPAMReconciler) reconcileDelete(ctx context.Context, cm *corev1.ConfigMap) (ctrl.Result, error) {
	logr := log.FromContext(ctx)
	logr.Info("Reconciling delete")
	if err := r.deleteCIDRPools(ctx); err != nil {
		return ctrl.Result{}, err
	}
	logr.Info("Removing finalizer")
	controllerutil.RemoveFinalizer(cm, spectrumXFinalizer)
	if err := r.Update(ctx, cm); err != nil {
		return ctrl.Result{}, fmt.Errorf("error while removing finalizer from configmap: %w", err)
	}
	return ctrl.Result{}, nil
}

// deleteCIDRPools deletes all the CIDR pools created by controller
func (r *NvIPAMReconciler) deleteCIDRPools(ctx context.Context) error {
	p := &unstructured.Unstructured{}
	p.SetGroupVersionKind(nvipamv1.GroupVersion.WithKind(cidrPoolType))
	if err := r.Client.DeleteAllOf(ctx, p, client.InNamespace(r.CIDRPoolsNamespace), client.MatchingLabels{
		spectrumXIPAMControllerName: "",
	}); err != nil {
		return fmt.Errorf("error while removing all %s: %w", p.GetObjectKind().GroupVersionKind().String(), err)
	}
	// Wait for all instances to be deleted
	timeout := 30 * time.Second
	interval := 1 * time.Second
	return wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx context.Context) (bool, error) {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(nvipamv1.GroupVersion.WithKind(cidrPoolType))

		if err := r.Client.List(ctx, list, client.InNamespace(r.CIDRPoolsNamespace), client.MatchingLabels{
			spectrumXIPAMControllerName: "",
		}); err != nil {
			return false, fmt.Errorf("error while listing %s: %w", list.GetObjectKind().GroupVersionKind().String(), err)
		}

		// If no resources remain, deletion is complete
		return len(list.Items) == 0, nil
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *NvIPAMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		Complete(r)
}
