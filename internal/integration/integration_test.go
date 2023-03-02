package integration

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/tidwall/gjson"
)

func retry(fn func() error, interval time.Duration, attempts int) error {
	err := fn()
	if err == nil {
		return nil
	}
	if attempts > 0 {
		time.Sleep(interval)
		return retry(fn, interval, attempts-1)
	} else {
		return err
	}
}

func command(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestClusterIP(t *testing.T) {
	if _, useExistingCluster := os.LookupEnv("USE_EXISTING_CLUSTER"); !useExistingCluster {
		t.Skip("set USE_EXISTING_CLUSTER and make sure kubectl can access it to run integration tests")
	}
	json, err := command("kubectl", "get", "nodes", "-ojson")
	if err != nil {
		t.Fatal(err)
	}
	nodes := gjson.Get(json, "items")

	out, err := command("kubectl", "apply", "-f", "../../config/samples/nodes-clusterip.yaml")
	if err != nil {
		t.Fatal(err, out)
	}

	err = retry(func() error {
		json, err := command("kubectl", "get", "clusterip", "clusterip-nodes", "-ojson")
		if err != nil {
			return err
		}
		status := gjson.Get(json, "status")
		if status.Get("state").String() != "Ready" {
			return fmt.Errorf("Expected status Ready, got %s", status)
		} else {
			numIPs := len(status.Get("nodeIPs").Array())
			numNodes := len(nodes.Array())
			if numNodes != numIPs {
				return fmt.Errorf("in the cluster with %v nodes %v IPs received", numNodes, numIPs)
			}
			for _, node := range status.Get("nodeIPs").Array() {
				t.Log("node:", node.Get("nodeLabel").String(), ",IP:", node.Get("ip"))
			}
		}
		return nil
	}, time.Second, 30)
	if err != nil {
		t.Fatal(err)
	}
}
