package controller

import (
	"context"
	"time"

	"github.com/kyma-project/cluster-ip/api/v1alpha1"
	"github.com/kyma-project/cluster-ip/internal/ip"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Controller", func() {
	ctx := context.Background()
	It("can get own IP address", func() {
		clusterIP := v1alpha1.ClusterIP{
			TypeMeta: v1.TypeMeta{APIVersion: "operator.kyma-project.io/v1alpha1",
				Kind: "ClusterIP"},
			ObjectMeta: v1.ObjectMeta{Namespace: "default", Name: "sample"}}
		Expect(k8sClient.Create(ctx, &clusterIP)).Should(Succeed())
		var createdIP v1alpha1.ClusterIP
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: "sample"}, &createdIP)
			if err == nil {
				for _, node := range createdIP.Status.NodeIPs {
					if node.NodeLabel == "zone-1" && ip.IsValidIP4(node.IP) {
						return true
					}
				}
			}
			return false
		}, time.Second*10, time.Millisecond*200).Should(BeTrue(), "Status should contain valid IP")
		Expect(createdIP.Spec.NodeSpreadLabel).Should(Equal("topology.kubernetes.io/zone"))
	})

})
