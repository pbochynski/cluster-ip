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
	"encoding/json"
	"io/ioutil"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kyma-project/cluster-ip/api/v1alpha1"
	operatorv1alpha1 "github.com/kyma-project/cluster-ip/api/v1alpha1"
)

// ClusterIPReconciler reconciles a ClusterIP object
type ClusterIPReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterIP object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ClusterIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var clusterIP v1alpha1.ClusterIP
	logger := log.FromContext(ctx)
	var result map[string]any
	resp, err := http.Get("http://ifconfig.me/all.json")
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	defer resp.Body.Close()
	logger.Info("Response", "status", resp.Status, "body", string(body))
	ip := result["ip_addr"]
	logger.Info("Response", "IP", ip)
	r.Get(ctx, req.NamespacedName, &clusterIP)
	clusterIP.Status.IP = 
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.ClusterIP{}).
		Complete(r)
}
