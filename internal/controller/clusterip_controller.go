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
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Scheme          *runtime.Scheme
	Zone            string
	SystemNamespace string
	ZonesIP         map[string]string
	StartTime       metav1.Time
}

func (r *ClusterIPReconciler) FindZonedPod(ctx context.Context, zone string) *corev1.Pod {
	var pods corev1.PodList
	logger := log.FromContext(ctx)
	logger.Info("FindPods", "SystemNamespace", r.SystemNamespace)
	err := r.List(ctx, &pods, client.InNamespace(r.SystemNamespace), client.MatchingLabels{"cluster-ip.operator.kyma-project.io/zone": zone})
	if err != nil {
		logger.Error(err, "Can't fetch pods", "err", err)
		return nil
	}
	if pods.Items != nil && len(pods.Items) > 0 {
		return &pods.Items[0]
	}
	return nil
}
func (r *ClusterIPReconciler) GetZones(ctx context.Context) []string {
	logger := log.FromContext(ctx)
	var nodes corev1.NodeList
	err := r.List(ctx, &nodes)

	var zones = map[string]bool{}
	var result []string
	if err == nil {
		for _, n := range nodes.Items {
			logger.Info("Nodes", "name", n.Name)
			zone := n.Labels["topology.kubernetes.io/zone"]
			if zone != "" {
				if !zones[zone] {
					zones[zone] = true
					result = append(result, n.Labels["topology.kubernetes.io/zone"])
				}
			}
		}
	} else {
		logger.Error(err, "Error, fetching nodes")
	}
	return result
}
func (r *ClusterIPReconciler) GetExtIP(ctx context.Context) string {
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
	return ip.(string)
}

func (r *ClusterIPReconciler) CreateZonedPod(ctx context.Context, zone string) *corev1.Pod {
	var pod = corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "cluster-ip-worker-pod-" + zone,
		Namespace: r.SystemNamespace,
		Labels:    map[string]string{"cluster-ip.operator.kyma-project.io/zone": zone},
	},
		Spec: corev1.PodSpec{Containers: []corev1.Container{corev1.Container{
			Name:  "worker",
			Image: "ghcr.io/pbochynski/cluster-ip:0.0.3.02140904",
			Args:  []string{"--zone", zone},
		}},
			NodeSelector:       map[string]string{"topology.kubernetes.io/zone": zone},
			ServiceAccountName: "cluster-ip-controller-manager",
		}}
	logger := log.FromContext(ctx)
	logger.Info("Creating zoned pod", "zone", zone)
	err := r.Create(ctx, &pod)
	if err != nil {
		logger.Error(err, "CreateZonedPod")
		return nil
	}
	return &pod
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=clusterips/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;delete
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
	var clusterIP v1alpha1.ClusterIP
	// var pod corev1.Pod
	logger := log.FromContext(ctx)

	err := r.Get(ctx, req.NamespacedName, &clusterIP)
	if r.Zone != "" { // worker pod
		ip := r.GetExtIP(ctx)
		found := false
		for i, z := range clusterIP.Status.Zones {

			if z.Zone == r.Zone {
				if z.IP == ip && z.LastUpdateTime.After(r.StartTime.Time) {
					logger.Info("Nothing to do", "zone", z, "ip", ip, "lastUpdate", z.LastUpdateTime, "startTime", r.StartTime)
					return ctrl.Result{}, nil // nothing to do, everything is up to date
				} else {
					zone := &clusterIP.Status.Zones[i]
					zone.IP = ip
					zone.LastUpdateTime = metav1.Now()
					found = true
					logger.Info("Updating", "zone", zone)
				}
				break
			}
		}
		if !found {
			logger.Info("Not found zone - creating", "zone", r.Zone)

			clusterIP.Status.Zones = append(clusterIP.Status.Zones,
				operatorv1alpha1.ZoneIP{
					Zone:           r.Zone,
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
	} else { // main controller
		zones := r.GetZones(ctx)
		allDone := true
		updateStatus := false
		for _, z := range zones {
			found := false

			for _, s := range clusterIP.Status.Zones {

				if s.Zone == z {
					found = true
					if validIP4(s.IP) && s.LastUpdateTime.After(r.StartTime.Time) {
						r.ZonesIP[z] = s.IP
						pod := r.FindZonedPod(ctx, z)
						r.Delete(ctx, pod)
					} else {
						allDone = false
					}
					break
				}
			}
			logger.Info("Reconciliation", "Zone", z, "found", found, "allDone", allDone, "cachedIP", r.ZonesIP[z])

			if r.ZonesIP[z] == "" {
				if r.FindZonedPod(ctx, z) == nil {
					r.CreateZonedPod(ctx, z)
				}
			}

			if !found {
				if r.ZonesIP[z] != "" {
					clusterIP.Status.Zones = append(clusterIP.Status.Zones, operatorv1alpha1.ZoneIP{Zone: z,
						IP:             r.ZonesIP[z],
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
	}

	return ctrl.Result{}, nil
}

func validIP4(ipAddress string) bool {
	ipAddress = strings.Trim(ipAddress, " ")

	re, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	if re.MatchString(ipAddress) {
		return true
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.ClusterIP{}).
		Complete(r)
}
