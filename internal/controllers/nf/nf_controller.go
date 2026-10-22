/*
Copyright 2026.

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

package nf

import (
	"context"
	"fmt"
	"net/netip"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	nfmgr "github.com/mantra6g/bluefield3-plugin/internal/managers/nf"
	nfcfgmgr "github.com/mantra6g/bluefield3-plugin/internal/managers/nfcfg"
	p4tmgr "github.com/mantra6g/bluefield3-plugin/internal/managers/p4target"
)

// NetworkFunctionReconciler reconciles a NetworkFunction object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	NFManager       nfmgr.Manager
	NFConfigManager       nfcfgmgr.Manager
	P4TargetManager p4tmgr.Manager
}

// +kubebuilder:rbac:groups=plugins.loom.io,resources=networkfunctions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=plugins.loom.io,resources=networkfunctions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=plugins.loom.io,resources=networkfunctions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NetworkFunction object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	logger := logf.FromContext(ctx)
	logger.Info("Reconciling NetworkFunction", "name", req.Name, "namespace", req.Namespace)

	netfunc := &corev1alpha1.NetworkFunction{}
	if err := r.Get(ctx, req.NamespacedName, netfunc); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("NetworkFunction resource not found. Deleting associated resources if any.")
			// _ = r.NFManager.EnsureAbsent(ctx, req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch NetworkFunction")
		return ctrl.Result{}, err
	}
	// check if network is configured (interfaces up on proxy and proxy <-> BF configured)
	r.P4TargetManager.GetNetworkConfiguredCondition()
	// assign IP to the NetworkFunction if missing (nf.AssignedIP, e.g. fd00::2/128)
	// update status based on real life

	var funcIP netip.Addr
	var err error
	if netfunc.Status.AssignedIP == "" {
		var err error
		funcIP, err = r.P4TargetManager.AllocateNetworkFunctionIP()
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to allocate Network Function IP: %w", err)
		}
	} else {
		funcIP, err = netip.ParseAddr(netfunc.Status.AssignedIP)
		if err != nil {
			logger.Error(nil, "Invalid IP address in NetworkFunction status", "assignedIP", netfunc.Status.AssignedIP)
			return ctrl.Result{}, fmt.Errorf("invalid IP address in NetworkFunction status: %s", netfunc.Status.AssignedIP)
		}
	}
	nfIP := netip.PrefixFrom(funcIP, r.P4TargetManager.GetNfCIDR().Bits())

	nfHandle := r.NFManager.EnsurePresent(ctx, netfunc, nfIP)
	_ = nfHandle.Status()
	// update status
	// err = r.updateStatus(netfunc, status, nfIP)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&corev1alpha1.NetworkFunction{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
				netfunc, ok := e.Object.(*corev1alpha1.NetworkFunction)
				if !ok {
					return false
				}
				return netfunc.Spec.TargetName == r.P4TargetManager.GetName()
			},
			UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
				oldNf, ok1 := e.ObjectOld.(*corev1alpha1.NetworkFunction)
				newNf, ok2 := e.ObjectNew.(*corev1alpha1.NetworkFunction)
				if !ok1 || !ok2 {
					return false
				}
				if oldNf.Generation == newNf.Generation {
					return false
				}
				return oldNf.Spec.TargetName == r.P4TargetManager.GetName() ||
					newNf.Spec.TargetName == r.P4TargetManager.GetName()
			},
			DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
				netfunc, ok := e.Object.(*corev1alpha1.NetworkFunction)
				if !ok {
					return false
				}
				return netfunc.Spec.TargetName == r.P4TargetManager.GetName()
			},
		}).
		Watches(&corev1alpha1.NetworkFunctionConfig{},
			handler.EnqueueRequestsFromMapFunc(r.mapNetworkFunctionConfigToRequests)).
		Named("networkfunction").
		Complete(r)
}

func (r *Reconciler) mapNetworkFunctionConfigToRequests(_ context.Context, obj client.Object) []reconcile.Request {
	referencedNFList, err := r.NFConfigManager.GetAllNetworkFunctionsUsingConfig(client.ObjectKeyFromObject(obj))
	if err != nil {
		return nil
	}
	requests := make([]reconcile.Request, len(referencedNFList))
	for i := range referencedNFList {
		requests[i] = reconcile.Request{}
	}
	return requests
}

func (r *Reconciler) updateStatus(networkFunction *corev1alpha1.NetworkFunction,
	depStatus nfmgr.DeploymentStatus, funcIP netip.Prefix) error {
	original := networkFunction.DeepCopy()
	networkFunction.Status = calculateStatus(networkFunction, depStatus, funcIP)
	return r.Status().Patch(context.Background(), networkFunction, client.MergeFrom(original))
}

func calculateStatus(netFunc *corev1alpha1.NetworkFunction,
	depStatus nfmgr.DeploymentStatus, funcIP netip.Prefix) corev1alpha1.NetworkFunctionStatus {
	// Calculate Phase
	var calculatedPhase corev1alpha1.NetworkFunctionPhase
	var readyCondition corev1alpha1.NetworkFunctionCondition
	switch depStatus.Phase {
	case nfmgr.PhaseReady:
		calculatedPhase = corev1alpha1.NetworkFunctionRunning
		readyCondition = corev1alpha1.NetworkFunctionCondition{
			Type:    corev1alpha1.NetworkFunctionReady,
			Status:  metav1.ConditionTrue,
			Reason:  "NetworkFunctionReady",
			Message: "The NetworkFunction is ready",
		}
	case nfmgr.PhaseFailed:
		calculatedPhase = corev1alpha1.NetworkFunctionFailed
		readyCondition = corev1alpha1.NetworkFunctionCondition{
			Type:    corev1alpha1.NetworkFunctionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "NetworkFunctionNotReady",
			Message: "The NetworkFunction is not ready",
		}
	default:
		calculatedPhase = corev1alpha1.NetworkFunctionPending
		readyCondition = corev1alpha1.NetworkFunctionCondition{
			Type:    corev1alpha1.NetworkFunctionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "NetworkFunctionNotReady",
			Message: "The NetworkFunction is not ready",
		}
	}

	status := corev1alpha1.NetworkFunctionStatus{
		AssignedIP:         funcIP.Addr().String(),
		ObservedGeneration: netFunc.Generation,
		Phase:              calculatedPhase,
	}
	// Copy conditions from old status
	status.Conditions = CopyConditions(&netFunc.Status)
	// Update or add the ready condition
	status.Conditions = UpdateNFCondition(&status, readyCondition)
	return status
}