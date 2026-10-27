/*
Copyright 2025.

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

package p4target

import (
	"context"
	"fmt"
	"time"

	"github.com/mantra6g/bluefield3-plugin/internal/controllers/lease"
	"github.com/mantra6g/bluefield3-plugin/internal/controllers/p4target/utils"

	v1alpha1 "github.com/mantra6g/bluefield3-plugin/api/v1alpha1"
	p4tmgr "github.com/mantra6g/bluefield3-plugin/internal/managers/p4target"
	net_types "github.com/mantra6g/bluefield3-plugin/networking"
	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"

	"github.com/go-logr/logr"
	/* v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors" */

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles a P4Target object
type Reconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	P4TargetManager p4tmgr.Manager
	PullInterval    time.Duration
	Log             logr.Logger
}

type ContainerizedP4TargetConfigRequest struct {
	ContainerID  string `json:"container_id" validate:"required"`
	P4TargetName string `json:"p4target_name" validate:"required"`
}
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	logger := r.Log
	logger.V(1).Info("Reconciling P4Target")

	target := &corev1alpha1.P4Target{}
	if err := r.Get(ctx, types.NamespacedName{Name: r.P4TargetManager.GetName()}, target); err != nil {
		/* if errors.IsNotFound(err) {
			err = r.createP4Target(ctx)
			if err != nil {
				logger.Error(err, "Failed to create P4Target")
				return ctrl.Result{}, err
			}
			logger.Info("P4Target created")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		} -> moved to operator*/
		logger.Error(err, "Failed to get P4Target")
		return ctrl.Result{}, err
	}

	/* 
	-> commenting out, because target Pod is only used to get the node name, 
	-> which we already specify in the bluefield3target
	-> might make the POD_NAME/POD_NAMESPACE ENVs workaround obsolete

	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	if podName == "" || podNamespace == "" {
		logger.Error(nil, "POD_NAME or POD_NAMESPACE environment variable is not set")
		panic("POD_NAME or POD_NAMESPACE environment variable is not set")
	}

	targetPod := &v1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: podNamespace}, targetPod); err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("P4Target Pod not found. Requeueing after 10 seconds.")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		logger.Error(err, "Failed to get the Driver Pod associated with this P4Target")
		return ctrl.Result{}, err
	} */
		
	// NfCIDR is allocated by the primary p4target reconciler in the IML manager
	if len(target.Spec.NfCIDR) == 0 {
		logger.V(1).Info("P4Target does not have any IP address. Requeueing after 10 seconds.")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// TargetDataplaneIPs are assigned by the IML daemon
	if len(target.Spec.TargetDataplaneIPs) == 0 {
		logger.V(1).Info("P4Target does not have any TargetDataplaneIPs. Requeueing after 10 seconds.")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if err := r.P4TargetManager.EnsureNetworkConfiguration(net_types.TargetNetConfig{
		NfCIDR: target.Spec.NfCIDR,
	}); err != nil {
		logger.Error(err, "Failed to configure network")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.updateP4TargetStatus(ctx, target)//, targetPod)
}

/* 
--> moved to bf3target_controller
func (r *Reconciler) createP4Target(ctx context.Context) error {
	target := &corev1alpha1.P4Target{
		ObjectMeta: metav1.ObjectMeta{
			Name: r.P4TargetManager.GetName(),
			Labels: map[string]string{
				corev1alpha1.P4TargetArchitectureLabel: "doca-bf3",
			},
		},
		Spec: corev1alpha1.P4TargetSpec{},
	}
	return r.Create(ctx, target)
} */


/* func (r *Reconciler) updateP4TargetStatus(ctx context.Context, target *corev1alpha1.P4Target, targetPod *v1.Pod) error {
	original := target.DeepCopy()
	target.Status = r.calculateP4TargetStatus(targetPod)
	if utils.StatusChanged(original, target) {
		return r.Status().Patch(ctx, target, client.MergeFrom(original))
	}
	return nil
}

func (r *Reconciler) calculateP4TargetStatus(targetPod *v1.Pod) corev1alpha1.P4TargetStatus {
	targetStatus := corev1alpha1.P4TargetStatus{}
	targetStatus.NodeName = "dtn-04" // targetPod.Spec.NodeName // TODO: get from Bluefield3target spec
	targetStatus.Capacity = r.P4TargetManager.GetCapacity()
	targetStatus.Allocatable = r.P4TargetManager.GetAllocatable()

	// targetStatus.TargetIPs = utils.FilterIPs(r.P4TargetManager.GetTargetIPs())
	targetStatus.DriverIPs = utils.FilterIPs(r.P4TargetManager.GetDriverIPs())

	occupiedCondition := r.P4TargetManager.GetOccupiedCondition()
	healthyCondition := r.P4TargetManager.GetHealthyCondition()
	netconfCondition := r.P4TargetManager.GetNetworkConfiguredCondition()
	readyCondition := r.P4TargetManager.GetReadyCondition(healthyCondition, netconfCondition)

	targetStatus.Conditions = []corev1alpha1.P4TargetCondition{}
	targetStatus.Conditions = append(targetStatus.Conditions, readyCondition)
	targetStatus.Conditions = append(targetStatus.Conditions, healthyCondition)
	targetStatus.Conditions = append(targetStatus.Conditions, netconfCondition)
	targetStatus.Conditions = append(targetStatus.Conditions, occupiedCondition)
	return targetStatus
} */
func (r *Reconciler) updateP4TargetStatus(ctx context.Context, target *corev1alpha1.P4Target) error {
	original := target.DeepCopy()
	bf3target, err := r.FetchBF3TargetOwner(ctx, target); if err != nil {
		return fmt.Errorf("failed to fetch BF3Target %w", err)
	}
	target.Status = r.calculateP4TargetStatus(bf3target)
	if utils.StatusChanged(original, target) {
		return r.Status().Patch(ctx, target, client.MergeFrom(original))
	}
	return nil
}

func (r *Reconciler) calculateP4TargetStatus(bf3target *v1alpha1.BlueField3Target) corev1alpha1.P4TargetStatus {
	targetStatus := corev1alpha1.P4TargetStatus{}
	targetStatus.Capacity = r.P4TargetManager.GetCapacity()
	targetStatus.Allocatable = r.P4TargetManager.GetAllocatable()
	targetStatus.NodeName = *bf3target.Spec.NodeName
	// targetStatus.TargetIPs = utils.FilterIPs(r.P4TargetManager.GetTargetIPs())
	targetStatus.DriverIPs = utils.FilterIPs(r.P4TargetManager.GetDriverIPs())

	occupiedCondition := r.P4TargetManager.GetOccupiedCondition()
	healthyCondition := r.P4TargetManager.GetHealthyCondition()
	netconfCondition := r.P4TargetManager.GetNetworkConfiguredCondition()
	readyCondition := r.P4TargetManager.GetReadyCondition(healthyCondition, netconfCondition)

	targetStatus.Conditions = []corev1alpha1.P4TargetCondition{}
	targetStatus.Conditions = append(targetStatus.Conditions, readyCondition)
	targetStatus.Conditions = append(targetStatus.Conditions, healthyCondition)
	targetStatus.Conditions = append(targetStatus.Conditions, netconfCondition)
	targetStatus.Conditions = append(targetStatus.Conditions, occupiedCondition)
	return targetStatus
}

func (r *Reconciler) FetchBF3TargetOwner(ctx context.Context, p4target *corev1alpha1.P4Target) (*v1alpha1.BlueField3Target, error) {
	for _, owner := range p4target.OwnerReferences {
		if owner.APIVersion == v1alpha1.GroupVersion.String() &&
			owner.Kind == "BlueField3Target" {

			var target v1alpha1.BlueField3Target
			err := r.Get(ctx, types.NamespacedName{
				Namespace: p4target.Namespace,
				Name:      owner.Name,
			}, &target)
			if err != nil {
				return nil, err
			}

			return &target, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch owneing bluefield3target")
}

func (r *Reconciler) Start(ctx context.Context) error {
	r.Log.V(1).Info("Starting P4Target controller")
	backoff := lease.InitialBackoff
	timer := time.NewTimer(0)  // run immediately
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			res, err := r.Reconcile(ctx)
			if err != nil {
				// failure -> apply backoff and retry
				timer.Reset(backoff)

				backoff *= 2
				if backoff > lease.MaxBackoff {
					backoff = lease.MaxBackoff
				}
				continue
			}
			// success -> reset backoff
			backoff = lease.InitialBackoff

			// decide next schedule
			next := r.PullInterval
			if !res.IsZero() {
				next = res.RequeueAfter
			}
			timer.Reset(next)
		}
	}
}
