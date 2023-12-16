/*
Copyright 2023.

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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scalerv1alpha1 "autoscaler-operator/api/v1alpha1"
)

// AutoScalerReconciler reconciles a AutoScaler object
type AutoScalerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	InitialReplicas *int32
}

//+kubebuilder:rbac:groups=scaler.sudiptob2.github.io,resources=autoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scaler.sudiptob2.github.io,resources=autoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scaler.sudiptob2.github.io,resources=autoscalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AutoScaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *AutoScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Obtain logger from the context
	logger := log.FromContext(ctx)

	// Create an instance of AutoScaler to store the fetched resource
	scaler := &scalerv1alpha1.AutoScaler{}

	// Fetch the AutoScaler resource using its namespaced name
	err := r.Get(ctx, req.NamespacedName, scaler)
	if err != nil {
		// Log an error if unable to fetch the AutoScaler resource
		logger.Error(err, "unable to fetch Scaler")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Extract relevant information from the AutoScaler resource
	name := scaler.Spec.Name
	startTime := scaler.Spec.StartTime
	endTime := scaler.Spec.EndTime
	replicas := int32(scaler.Spec.Replicas)
	currentHour := time.Now().Hour()
	deployment := &v1.Deployment{}

	// Log information about the AutoScaler object
	logger.Info("Scaler object", "name", name, "startTime", startTime, "endTime", endTime, "replicas", replicas)

	// Fetch the Deployment associated with the AutoScaler
	err = r.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: "default",
	},
		deployment,
	)
	if err != nil {
		// Log an error if unable to fetch the associated Deployment
		logger.Error(err, "unable to fetch deployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If InitialReplicas is not set, store the initial replicas from the Deployment
	if r.InitialReplicas == nil {
		r.InitialReplicas = deployment.Spec.Replicas
	}

	// Check if the current hour is within the specified time range
	if currentHour >= startTime && currentHour <= endTime {
		// If replicas in the Deployment differ from the desired replicas, update the Deployment
		if *deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = &replicas
			err = r.Update(ctx, deployment)
			if err != nil {
				// Log an error if unable to update the Deployment
				logger.Error(err, "unable to update deployment")
				return ctrl.Result{}, err
			}
		}
	} else {
		// If outside the specified time range, revert to the initial number of replicas
		if *deployment.Spec.Replicas != *r.InitialReplicas {
			deployment.Spec.Replicas = r.InitialReplicas
			err = r.Update(ctx, deployment)
			if err != nil {
				// Log an error if unable to update the Deployment
				logger.Error(err, "unable to update deployment")
				return ctrl.Result{}, err
			}
		}
	}

	// Return a result indicating a requeue after 5 seconds
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalerv1alpha1.AutoScaler{}).
		Complete(r)
}
