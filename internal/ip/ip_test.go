package ip

import "testing"

func TestParallelIP(t *testing.T) {
	ip, err := GetIP(4)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf(ip)
}
