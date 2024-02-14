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
	"fmt"
	"os"

	"hash/crc32"
	s "strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kyma-project/cluster-ip/api/v1alpha1"
	operatorv1alpha1 "github.com/kyma-project/cluster-ip/api/v1alpha1"
	"github.com/kyma-project/cluster-ip/internal/ip"
)

// ClusterIPReconciler reconciles a ClusterIP object
type ClusterIPReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Node            string
	NodeSpreadLabel string
	SystemNamespace string
	NodeIP          map[string]string
	StartTime       metav1.Time
}

func (r *ClusterIPReconciler) MyImageName(ctx context.Context) string {
	logger := log.FromContext(ctx)
	imageName := os.Getenv("IMAGE_NAME")
	var pod corev1.Pod
	if imageName == "" {
		ns := os.Getenv("MY_POD_NAMESPACE")
		name := os.Getenv("MY_POD_NAME")
		err := r.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &pod)
		if err != nil {
			logger.Error(err, "Can't find my pod", "ns", ns, "pod", name)
			panic(err)
		}
		for _, c := range pod.Spec.Containers {
			if s.Contains(c.Image, "cluster-ip") {
				imageName = c.Image
				logger.Info("Found my container", "image", c.Image)
				break
			}
		}
	}
	if imageName == "" {
		err := fmt.Errorf("cannot find controller image")
		panic(err)
	}
	return imageName
}
func hash(input string) string {
	crc32q := crc32.MakeTable(0xD5828281)
	return fmt.Sprintf("%08x", crc32.Checksum([]byte(input), crc32q))
}
func (r *ClusterIPReconciler) FindZonedPod(ctx context.Context, zone string) *corev1.Pod {
	var pods corev1.PodList
	logger := log.FromContext(ctx)
	err := r.List(ctx, &pods, client.InNamespace(r.SystemNamespace), client.MatchingLabels{"cluster-ip.operator.kyma-project.io/zone": hash(zone)})
	if err != nil {
		logger.Error(err, "Can't fetch pods", "err", err)
		return nil
	}
	if pods.Items != nil && len(pods.Items) > 0 {
		return &pods.Items[0]
	}
	return nil
}
func (r *ClusterIPReconciler) GetNodeLabels(ctx context.Context, nodeSpreadLabel string) []string {
	logger := log.FromContext(ctx)
	var nodes corev1.NodeList
	err := r.List(ctx, &nodes)

	var zones = map[string]bool{}
	var result []string
	if err == nil {
		for _, n := range nodes.Items {
			if len(n.Spec.Taints) > 0 {
				continue
			}
			zone := n.Labels[nodeSpreadLabel]
			if zone != "" {
				if !zones[zone] {
					zones[zone] = true
					result = append(result, n.Labels[nodeSpreadLabel])
				}
			}
		}
	} else {
		logger.Error(err, "Error, fetching nodes")
	}
	return result
}

func (r *ClusterIPReconciler) CreateOrUpdatePod(ctx context.Context, label string, nodeSpreadLabel string, image string) *corev1.Pod {
	existingPod := r.FindZonedPod(ctx, label)
	podTemplate := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "cluster-ip-worker-pod-" + hash(label),
		Namespace: r.SystemNamespace,
		Labels:    map[string]string{"cluster-ip.operator.kyma-project.io/zone": hash(label)},
	},
		Spec: corev1.PodSpec{Containers: []corev1.Container{corev1.Container{
			Name:  "worker",
			Image: image,
			Args:  []string{"--node", label, "--nodeSpreadLabel", nodeSpreadLabel},
		}},
			NodeSelector:       map[string]string{nodeSpreadLabel: label},
			ServiceAccountName: "cluster-ip-controller-manager",
		}}
	logger := log.FromContext(ctx)
	if existingPod == nil {
		logger.Info("Creating new pod for node label", "label", label)
		if err := r.Create(ctx, podTemplate); err != nil {
			logger.Error(err, "Can't create pod")
			return nil
		}
		return podTemplate
	} else {
		logger.Info("Updating pod for node label", "label", label)
		for i := range existingPod.Spec.Containers {
			c := &existingPod.Spec.Containers[i]
			if c.Name == "worker" {
				c.Image = image
				c.Args = []string{"--node", label, "--nodeSpreadLabel", nodeSpreadLabel}
			}
		}
		if err := r.Update(ctx, existingPod); err != nil {
			logger.Error(err, "Can't update pod")
			return nil
		}
		return existingPod
	}
}
func (r *ClusterIPReconciler) ReconcileWorker(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var clusterIP v1alpha1.ClusterIP
	logger := log.FromContext(ctx)

	err := r.Get(ctx, req.NamespacedName, &clusterIP)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if r.NodeSpreadLabel != clusterIP.Spec.NodeSpreadLabel {
		logger.Info("Skip reconciliation", "node", r.Node, "nodeSpreadLabel", r.NodeSpreadLabel, "clusterIP", clusterIP)
		return ctrl.Result{}, nil
	}

	ip, err := ip.GetIP(2)
	if err != nil {
		return ctrl.Result{}, err
	}
	found := false
	for i, z := range clusterIP.Status.NodeIPs {

		if z.NodeLabel == r.Node {
			if z.IP == ip && z.LastUpdateTime.After(r.StartTime.Time) {
				logger.Info("Nothing to do", "zone", z, "ip", ip, "lastUpdate", z.LastUpdateTime, "startTime", r.StartTime)
				return ctrl.Result{}, nil // nothing to do, everything is up to date
			} else {
				node := &clusterIP.Status.NodeIPs[i]
				node.IP = ip
				node.LastUpdateTime = metav1.Now()
				found = true
				logger.Info("Updating", "node", node)
			}
			break
		}
	}
	if !found {
		logger.Info("Not found label - creating", "label", r.Node)

		clusterIP.Status.NodeIPs = append(clusterIP.Status.NodeIPs,
			operatorv1alpha1.NodeIP{
				NodeLabel:      r.Node,
				IP:             ip,
				LastUpdateTime: metav1.Now()})
	}
	if clusterIP.Status.State == "" {
		clusterIP.Status.State = "Processing"
	}
	logger.Info("Status update started", "status", clusterIP.Status)
	err = r.Status().Update(ctx, &clusterIP)
	if err != nil {
		logger.Error(err, "Can't update status", "err", err)
	}

	return ctrl.Result{}, nil
}
func (r *ClusterIPReconciler) NodeWatcherToRequests(ctx context.Context, node client.Object) []reconcile.Request {
	var clusterIPs v1alpha1.ClusterIPList
	err := r.List(ctx, &clusterIPs)
	if err != nil {
		return []reconcile.Request{}
	}
	logger := log.FromContext(ctx)
	logger.Info("NodeWatcher invoked", "node", node.GetName(), "clusterIP count", len(clusterIPs.Items))
	requests := make([]reconcile.Request, len(clusterIPs.Items))
	for i, item := range clusterIPs.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch

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
	if r.Node != "" { // worker pod
		return r.ReconcileWorker(ctx, req)
	}

	var clusterIP v1alpha1.ClusterIP

	logger := log.FromContext(ctx)
	err := r.Get(ctx, req.NamespacedName, &clusterIP)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	image := r.MyImageName(ctx)
	zones := r.GetNodeLabels(ctx, clusterIP.Spec.NodeSpreadLabel)
	logger.Info("Reconciliation", "cr", clusterIP.Name, "label", clusterIP.Spec.NodeSpreadLabel, "values", zones)
	allDone := true
	updateStatus := false
	for _, z := range zones {
		found := false

		for _, s := range clusterIP.Status.NodeIPs {

			if s.NodeLabel == z {
				found = true
				if ip.IsValidIP4(s.IP) && s.LastUpdateTime.After(r.StartTime.Time) {
					r.NodeIP[z] = s.IP
					pod := r.FindZonedPod(ctx, z)
					r.Delete(ctx, pod)
				} else {
					allDone = false
				}
				break
			}
		}

		if r.NodeIP[z] == "" {
			r.CreateOrUpdatePod(ctx, z, clusterIP.Spec.NodeSpreadLabel, image)
		}

		if !found {
			if r.NodeIP[z] != "" {
				clusterIP.Status.NodeIPs = append(clusterIP.Status.NodeIPs, operatorv1alpha1.NodeIP{NodeLabel: z,
					IP:             r.NodeIP[z],
					LastUpdateTime: metav1.Now()})
				updateStatus = true
			} else {
				allDone = false
			}
		}
	}
	if clusterIP.Status.State == "" {
		clusterIP.Status.State = "Processing"
		updateStatus = true
	}
	if allDone && clusterIP.Status.State != "Ready" {
		clusterIP.Status.State = "Ready"
		updateStatus = true
	}
	if updateStatus {
		err = r.Status().Update(ctx, &clusterIP)
		if err != nil {
			logger.Error(err, "Can't update status", "err", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.ClusterIP{}).
		Watches(&corev1.Node{}, handler.EnqueueRequestsFromMapFunc(r.NodeWatcherToRequests), builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}
