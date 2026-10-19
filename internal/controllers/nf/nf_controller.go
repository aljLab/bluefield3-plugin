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
	"net/netip"
	// "path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	// dplclient "github.com/mantra6g/bluefield3-plugin/client"
	nfmgr "github.com/mantra6g/bluefield3-plugin/internal/managers/nf"
)

// NetworkFunctionReconciler reconciles a NetworkFunction object
type NetworkFunctionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// DplClient *dplclient.DplClient
	NFManager nfmgr.Manager
}

// +kubebuilder:rbac:groups=my.domain,resources=networkfunctions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=my.domain,resources=networkfunctions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=my.domain,resources=networkfunctions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NetworkFunction object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *NetworkFunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
	// update status based on real life

	/* // fetch program
	fileURL := netfunc.Spec.P4File
	err := r.DplClient.FetchProgram(fileURL); if err != nil {
		logger.Error(err, "unable to fetch p4 program")
		return ctrl.Result{}, err
	}

	// send compile request
	err = r.DplClient.CompileP4File(filepath.Base(fileURL)); if err != nil {
		logger.Error(err, "unable to compile p4 program")
		return ctrl.Result{}, err
	}

	// send deploy request
	err = r.DplClient.LoadP4RuntimeShell(filepath.Base(fileURL)); if err != nil {
		logger.Error(err, "unable to load p4 program")
		return ctrl.Result{}, err
	} */

	nfHandle := r.NFManager.EnsurePresent(ctx, netfunc, netip.MustParsePrefix("10.0.0.1/24"))
	_ = nfHandle.Status()
	// err = r.updateStatus(netfunc, status, fullNfAddress)
	// update status
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkFunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// TODO: index targetname field?
		For(&corev1alpha1.NetworkFunction{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
				return true
				// netfunc, ok := e.Object.(*corev1alpha1.NetworkFunction)
				// if !ok {
				//	return false
				// }
				// return netfunc.Spec.TargetName == "bluefield3-host" // change later to generic name
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
				return oldNf.Spec.TargetName == "bluefield3-host" || // change later to generic name
					newNf.Spec.TargetName == "bluefield3-host"
			},
			DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
				netfunc, ok := e.Object.(*corev1alpha1.NetworkFunction)
				if !ok {
					return false
				}
				return netfunc.Spec.TargetName == "bluefield3-host" // change later to generic
			},
		}).
		// Watches(&corev1alpha1.NetworkFunctionConfig{},
		//	handler.EnqueueRequestsFromMapFunc(r.mapNetworkFunctionConfigToRequests)).
		Named("networkfunction").
		Complete(r)
}
