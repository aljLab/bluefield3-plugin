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

package bf3

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/mantra6g/bluefield3-plugin/internal/controllers/bf3/util"
	bf3util "github.com/mantra6g/bluefield3-plugin/internal/controllers/bf3/util"
	"github.com/mantra6g/bluefield3-plugin/api/v1alpha1"
	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
)

// BlueField3TargetReconciler reconciles a BlueField3Target object
type BlueField3TargetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *util.Bf3DriverConfig
}

// +kubebuilder:rbac:groups=plugins.loom.io,resources=bluefield3targets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=plugins.loom.io,resources=bluefield3targets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=plugins.loom.io,resources=bluefield3targets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.loom.io,resources=p4targets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BlueField3Target object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *BlueField3TargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.V(1).Info("Reconciling BlueField3Target")
	
	bf3Target := &v1alpha1.BlueField3Target{}
	err := r.Get(ctx, req.NamespacedName, bf3Target)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("BlueField3Target resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch BlueField3Target")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ensure P4Target deployment
	err = r.createP4Target(ctx, bf3Target); if err != nil {
		logger.Error(err, "unable to create P4Target")
		return ctrl.Result{}, err
	}
	
	// deploy driver pod
	_, err = r.ensureDeployment(ctx, bf3Target)
	if err != nil {
		logger.Error(err, "failed to ensure bf3Target has correct defaults and finalizers")
		return ctrl.Result{}, err
	}
	// update status

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BlueField3TargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BlueField3Target{}).
		Owns(&corev1alpha1.P4Target{}).
		Owns(&appsv1.Deployment{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("infra-bluefield3target").
		Complete(r)
}

func (r *BlueField3TargetReconciler) ensureDeployment(ctx context.Context,
	target *v1alpha1.BlueField3Target) (*appsv1.Deployment, error) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-driver", target.Name),
			Namespace: r.Config.PodNamespace,
		},
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, dep, func() error {
		dep.Labels = bf3util.EnsureBf3DeploymentLabels(target, dep.Labels)
		dep.Annotations = bf3util.EnsureBf3DeploymentAnnotations(target, dep.Annotations)
		dep.Finalizers = bf3util.EnsureBf3DeploymentFinalizers(target, dep.Finalizers)
		dep.Spec = *bf3util.EnsureBf3DeploymentSpec(target, &dep.Spec, r.Config)
		return controllerutil.SetControllerReference(target, dep, r.Scheme) // BlueField3Target owns Deployment
	})
	return dep, err
}

func (r *BlueField3TargetReconciler) createP4Target(ctx context.Context, bf3target *v1alpha1.BlueField3Target) error {
	target := &corev1alpha1.P4Target{
		ObjectMeta: metav1.ObjectMeta{
			Name: *bf3target.Spec.NodeName,
			Labels: map[string]string{
				corev1alpha1.P4TargetArchitectureLabel: "doca-bf3",
			},
		},
		Spec: *bf3util.EnsureP4TargetSpec(bf3target),
	}
	return r.Create(ctx, target)
}
