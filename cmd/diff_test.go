package cmd

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/base/dsfs"
)

func TestDiffComplete(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	cases := []struct {
		args []string
		err  string
	}{
		{[]string{}, ""},
		{[]string{"one arg"}, ""},
		{[]string{"one arg", "two args"}, ""},
	}

	for i, c := range cases {
		opt := &DiffOptions{
			IOStreams: streams,
		}

		opt.Complete(f, c.args)

		if c.err != errs.String() {
			t.Errorf("case %d, error mismatch. Expected: '%s', Got: '%s'", i, c.err, errs.String())
			ioReset(in, out, errs)
			continue
		}

		if opt.DatasetRequests == nil {
			t.Errorf("case %d, opt.DatasetRequests not set.", i)
			ioReset(in, out, errs)
			continue
		}
		ioReset(in, out, errs)
	}
}

func TestDiffRun(t *testing.T) {
	streams, in, out, errs := ioes.NewTestIOStreams()
	setNoColor(true)

	// to keep hashes consistent, artificially specify the timestamp by overriding
	// the dsfs.Timestamp func
	prev := dsfs.Timestamp
	defer func() { dsfs.Timestamp = prev }()
	dsfs.Timestamp = func() time.Time { return time.Date(2001, 01, 01, 01, 01, 01, 01, time.UTC) }

	f, err := NewTestFactory()
	if err != nil {
		t.Errorf("error creating new test factory: %s", err)
		return
	}

	good := []struct {
		description string
		opt         *DiffOptions
		stdout      string
	}{
		{"diff two dataset metas",
			&DiffOptions{
				Refs:     NewListOfRefSelects([]string{"me/movies", "me/cities"}),
				Selector: "meta",
			},
			`0 elements. 1 insert. 1 delete.

 qri: "md:0"
-title: "example movie data"
+title: "example city data"
`,
		},
		{"diff json output",
			&DiffOptions{
				Refs:     NewListOfRefSelects([]string{"me/movies", "me/cities"}),
				Selector: "meta",
				Format:   "json",
			},
			`{"stat":{"leftNodes":3,"rightNodes":3,"leftWeight":45,"rightWeight":43,"inserts":1,"deletes":1},"diff":[[" ","qri","md:0"],["-","title","example movie data"],["+","title","example city data"]]}
`,
		},
	}

	for _, c := range good {
		t.Run(c.description, func(t *testing.T) {
			dsr, err := f.DatasetRequests()
			if err != nil {
				t.Fatalf("case %s, error creating dataset request: %s", c.description, err)
			}

			opt := c.opt
			opt.IOStreams = streams
			opt.DatasetRequests = dsr

			if err = opt.Run(); err != nil {
				t.Fatalf("case %s unexpected error: %s", c.description, err)
			}

			if diff := cmp.Diff(out.String(), c.stdout); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}

			ioReset(in, out, errs)
		})
	}

	bad := []struct {
		opt *DiffOptions
		err string
	}{
		{
			&DiffOptions{},
			"nothing to diff",
		},
	}

	for _, c := range bad {
		dsr, err := f.DatasetRequests()
		if err != nil {
			t.Errorf("case %s, error creating dataset request: %s", c.err, err)
			continue
		}

		opt := c.opt
		opt.Refs = NewListOfRefSelects([]string{})
		opt.IOStreams = streams
		opt.DatasetRequests = dsr

		err = opt.Run()

		if err == nil {
			t.Errorf("expected: '%s', got no error", c.err)
			ioReset(in, out, errs)
			continue
		}
		if c.err != err.Error() {
			t.Errorf("error mismatch. expected: '%s', got: '%s'", c.err, err.Error())
		}
		ioReset(in, out, errs)
	}
}
