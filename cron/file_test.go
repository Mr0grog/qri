package cron

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFsJobStore(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "TestFsJobStore")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	newStore := func() JobStore {
		return NewFileJobStore(filepath.Join(tmp, "jobs.cbor"))
	}
	RunJobStoreTests(t, newStore)
}

func BenchmarkFileJobStore(b *testing.B) {
	js := make(jobs, 1000)
	for i := range js {
		js[i] = &Job{
			Name:        fmt.Sprintf("job_%d", i),
			Type:        JTDataset,
			Periodicity: mustRepeatingInterval("R/P1H"),
		}
	}

	tmp, err := ioutil.TempDir(os.TempDir(), "BenchmarkFbJobStore")
	if err != nil {
		b.Fatal(err)
	}
	
	defer os.RemoveAll(tmp)
	store := NewFileJobStore(filepath.Join(tmp, "jobs.cbor"))

	for i := 0; i < b.N; i++ {
		if err := store.PutJobs(js...); err != nil {
			b.Fatal(err)
		}
		if _, err := store.Jobs(0, 0); err != nil {
			b.Fatal(err)
		}
	}
}