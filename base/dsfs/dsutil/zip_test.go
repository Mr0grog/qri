package dsutil

import (
	"archive/zip"
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/dataset"
	"github.com/qri-io/qri/base/dsfs"
)

func TestWriteZipArchive(t *testing.T) {
	ctx := context.Background()
	store, names, err := testStore()
	if err != nil {
		t.Errorf("error creating store: %s", err.Error())
		return
	}

	ds, err := dsfs.LoadDataset(ctx, store, names["movies"])
	if err != nil {
		t.Errorf("error fetching movies dataset from store: %s", err.Error())
		return
	}

	buf := &bytes.Buffer{}
	if err = WriteZipArchive(ctx, store, ds, "yaml", "peer/ref@a/ipfs/b", buf); err != nil {
		t.Errorf("error writing zip archive: %s", err.Error())
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Errorf("error creating zip reader: %s", err.Error())
		return
	}

	// TODO (dlong): Actually test the contents of the zip.
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Errorf("error opening file %s in package", f.Name)
			break
		}

		if err := rc.Close(); err != nil {
			t.Errorf("error closing file %s in package", f.Name)
			break
		}
	}
}

func TestWriteZipArchiveFullDataset(t *testing.T) {
	ctx := context.Background()
	store, names, err := testStoreWithVizAndTransform()
	if err != nil {
		t.Errorf("error creating store: %s", err.Error())
		return
	}

	ds, err := dsfs.LoadDataset(ctx, store, names["movies"])
	if err != nil {
		t.Errorf("error fetching movies dataset from store: %s", err.Error())
		return
	}

	_, err = store.Get(ctx, names["transform_script"])
	if err != nil {
		t.Errorf("error fetching movies dataset from store: %s", err.Error())
		return
	}

	buf := &bytes.Buffer{}
	if err = WriteZipArchive(ctx, store, ds, "json", "peer/ref@a/ipfs/b", buf); err != nil {
		t.Errorf("error writing zip archive: %s", err.Error())
		return
	}

	tmppath := filepath.Join(os.TempDir(), "exported.zip")
	// defer os.RemoveAll(tmppath)
	t.Log(tmppath)
	err = ioutil.WriteFile(tmppath, buf.Bytes(), os.ModePerm)
	if err != nil {
		t.Errorf("error writing temp zip file: %s", err.Error())
		return
	}

	expectFile := testdataFile("testdata/zip/exported.zip")
	expectBytes, err := ioutil.ReadFile(expectFile)
	if err != nil {
		t.Errorf("error reading expected bytes: %s", err.Error())
		return
	}
	if diff := cmp.Diff(expectBytes, buf.Bytes()); diff != "" {
		t.Errorf("byte mismatch (-want +got):\n%s", diff)
	}
}

func TestUnzipDatasetBytes(t *testing.T) {
	path := testdataFile("testdata/zip/exported.zip")
	zipBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	dsp := &dataset.Dataset{}
	if err := UnzipDatasetBytes(zipBytes, dsp); err != nil {
		t.Error(err)
	}
}
func TestUnzipDataset(t *testing.T) {
	if err := UnzipDataset(bytes.NewReader([]byte{}), 0, &dataset.Dataset{}); err == nil {
		t.Error("expected passing bad reader to error")
	}

	path := testdataFile("testdata/zip/exported.zip")
	zipBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	dsp := &dataset.Dataset{}
	if err := UnzipDataset(bytes.NewReader(zipBytes), int64(len(zipBytes)), dsp); err != nil {
		t.Error(err)
	}
}
func TestUnzipGetContents(t *testing.T) {
	if _, err := UnzipGetContents([]byte{}); err == nil {
		t.Error("expected passing bad reader to error")
	}

	path := testdataFile("testdata/zip/exported.zip")
	zipBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	res, err := UnzipGetContents(zipBytes)
	if err != nil {
		t.Error(err)
	}
	expectLen := 6
	// files include:
	// dataset.json
	// body.csv
	// index.html
	// ref.txt
	// transform.star
	// viz.html
	if len(res) != expectLen {
		t.Errorf("contents length mismatch. expected: %d, got: %d", expectLen, len(res))
	}
}