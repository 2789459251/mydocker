package subsystems

import (
	"os"
	"testing"
)

func TestMemoryCgroup(t *testing.T) {
	memSubSys := MemorySubSystem{}
	resConfig := ResourceConfig{
		MemoryLimit: "1000m",
	}
	testCgroup := "testmemlimit"

	if err := memSubSys.Set(testCgroup, &resConfig); err != nil {
		t.Fatalf("cgroup fail %v", err)
	}

	res := &ResourceConfig{MemoryLimit: "10m"}

	if err := memSubSys.Apply(testCgroup, os.Getpid(), res); err != nil {
		t.Fatalf("cgroup Apply %v", err)
	}

	if err := memSubSys.Remove(testCgroup); err != nil {
		t.Fatalf("cgroup remove %v", err)
	}
}
