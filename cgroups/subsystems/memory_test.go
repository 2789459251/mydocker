package subsystems

import (
	"os"
	"testing"
)

func TestMemoryCgroup(t *testing.T) {
	memSubSys := MemorySubsystem{}
	resConfig := ResourceConfig{
		MemoryLimit: "1000m",
	}
	testCgroup := "testmemlimit"

	if err := memSubSys.Set(testCgroup, &resConfig); err != nil {
		t.Fatalf("cgroup fail %v", err)
	}

	if err := memSubSys.Apply(testCgroup, os.Getpid()); err != nil {
		t.Fatalf("cgroup Apply %v", err)
	}

	if err := memSubSys.Remove(testCgroup); err != nil {
		t.Fatalf("cgroup remove %v", err)
	}
}
