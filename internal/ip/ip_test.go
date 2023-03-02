package ip

import (
	"testing"
)

func TestValidIP(t *testing.T) {
	clusterIp, err := GetIP(4)
	if err != nil {
		t.Error(err)
	}
	if !IsValidIP4(clusterIp) {
		t.Errorf("'%s' expected to be valid IP", clusterIp)
	}
}
