package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	golog "github.com/ipfs/go-log"
	"github.com/qri-io/qri/config"
	"github.com/qri-io/qri/startf"
)

func init() {
	// TODO (b5) - ask go-ipfs folks if the shutdown messages can be INFO level
	// instead of error level to avoid:
	// 10:12:42.396 ERROR       core: core is shutting down...
	// after all sorts of tests
	golog.SetLogLevel("core", "CRITICAL")
}

// ioReset resets the in, out, errs buffers
// convenience function used in testing
func ioReset(in, out, errs *bytes.Buffer) {
	in.Reset()
	out.Reset()
	errs.Reset()
}

func confirmQriNotRunning() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.DefaultAPIPort))
	if err != nil {
		return fmt.Errorf("it looks like a qri server is already running on port %d, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
}

func confirmUpdateServiceNotRunning() error {
	l, err := net.Listen("tcp", config.DefaultUpdateAddress)
	if err != nil {
		return fmt.Errorf("it looks like a qri update service is already running on port %d, please close before running tests", config.DefaultAPIPort)
	}

	l.Close()
	return nil
}

const moviesCSVData = `movie_title,duration
Avatar,178
Pirates of the Caribbean: At World's End,169
Spectre,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens,15
John Carter,132
Spider-Man 3,156
Tangled,100
Avengers: Age of Ultron,141`

const moviesCSVData2 = `movie_title,duration
Avatar,178
Pirates of the Caribbean: At World's End,169
Spectre,148
The Dark Knight Rises ,164
Star Wars: Episode VII - The Force Awakens,15
John Carter,132
Spider-Man 3,156
Tangled,100
Avengers: Age of Ultron,141
A Wild Film Appears!,2000
Another Film!,121`

const linksJSONData = `[
  "http://datatogether.org",
  "https://datatogether.org/css/style.css",
  "https://datatogether.org/img/favicon.ico",
  "https://datatogether.org",
  "https://datatogether.org/public-record",
  "https://datatogether.org/activities",
  "https://datatogether.org/activities/harvesting",
  "https://datatogether.org/activities/monitoring",
  "https://datatogether.org/activities/storing",
  "https://datatogether.org/activities/rescuing",
  "http://2017.code4lib.org",
  "https://datatogether.org/presentations/Code4Lib%202017%20-%20Golden%20Age%20for%20Libraries%20-%20Storing%20Data%20Together.pdf",
  "https://datatogether.org/presentations/Code4Lib%202017%20-%20Golden%20Age%20for%20Libraries%20-%20Storing%20Data%20Together.key",
  "http://www.esipfed.org/meetings/upcoming-meetings/esip-summer-meeting-2017",
  "https://datatogether.org/presentations/Data%20Together%20-%20ESIP%20Summer%20Meeting%20July%202017.pdf",
  "https://datatogether.org/presentations/Data%20Together%20-%20ESIP%20Summer%20Meeting%20July%202017.key",
  "https://archive.org/details/ndsr-dc-2017",
  "https://datatogether.org/presentations/Data%20Together%20-%20NDSR%20-%20swadeshi.pdf",
  "https://datatogether.org/presentations/Data%20Together%20-%20NDSR%20-%20swadeshi.key",
  "https://github.com/datatogether"
]`

const profileData = `
{
	"description" : "I'm a description!"
}
`

// Test that saving a dataset with a relative body path works, and validate the contents of that
// body match what was given to the save command.
func TestSaveRelativeBodyPath(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_relative_body")
	defer run.Delete()

	// Save a dataset which has a body as a relative path
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_movies")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.csv")

	// Read the body from the testdata input file.
	f, _ := os.Open("testdata/movies/body_ten.csv")
	expectBytes, _ := ioutil.ReadAll(f)
	expectBody := string(expectBytes)

	// Make sure they match.
	if actualBody != expectBody {
		t.Errorf("error reading body, expect \"%s\", actual \"%s\"", actualBody, expectBody)
	}
}

// Test that saving three revisions, then removing the newest two, leaves the first body.
func TestRemoveOnlyTwoRevisions(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_remove_only_two_revisions")
	defer run.Delete()

	// Save three revisions, then remove two
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --revisions=2")

	// Read body from the dataset that was saved.
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.csv")

	// Read the body from the testdata input file.
	f, _ := os.Open("testdata/movies/body_ten.csv")
	expectBytes, _ := ioutil.ReadAll(f)
	expectBody := string(expectBytes)

	// Make sure they match.
	if expectBody != actualBody {
		t.Errorf("error reading body, expect \"%s\", actual \"%s\"", expectBody, actualBody)
	}
}

// Test that adding three revision, then removing all of them leaves nothing.
func TestRemoveAllRevisionsLongForm(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_remove_only_one_revision")
	defer run.Delete()

	// Save three versions, then remove all of them.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --revisions=all")

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := run.GetPathForDataset(t, 0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that adding three revision, then removing all of them leaves nothing, using --all.
func TestRemoveAllRevisionsShortForm(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_remove_only_one_revision")
	defer run.Delete()

	// Save three versions, then remove all of them, using the --all flag.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	run.MustExec(t, "qri remove me/test_movies --all")

	// Read path for dataset, which shouldn't exist anymore.
	dsPath := run.GetPathForDataset(t, 0)
	if dsPath != "" {
		t.Errorf("expected dataset to be removed entirely, found at \"%s\"", dsPath)
	}
}

// Test that save can override a single component, meta in this case.
func TestSaveThenOverrideMetaComponent(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save a version, then save another with a new meta component.
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	actual := run.DatasetMarshalJSON(t, dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override.yaml.
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"meta:\n\tupdated title","path":"/ipfs/QmeYjmGoCVkrkn95KWq6fcs4Y5JcHcKHvSSkXTskidoNod","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"meta updated title"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmYvBrUG2F4gr8a63EdikKxQU7NZ5SkqFAFGhMbNNHeNUy","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that saving with two components at once will merge them together.
func TestSaveTwoComponents(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save a version, then same another with two components at once
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/structure_override.json me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	actual := run.DatasetMarshalJSON(t, dsPath)

	// This dataset is ds_ten.yaml, with the meta replaced by meta_override ("different title") and
	// the structure replaced by structure_override (lazyQuotes: false && title: "name").
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"meta:\n\tupdated title\nstructure:\n\tupdated formatConfig.lazyQuotes\n\tupdated schema.items.items.0.title","path":"/ipfs/Qmf51CD3zW64ffoWja32bKh3BSyMwvMSbh9A8PtrA7fDJi","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"updated meta and structure"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmcSJnaS6xwcJXMKS3SPbyCnvorTefpMeMMJeXyMfxtTq8","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":false},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"name","type":"string"},{"title":"duration","type":"integer"}]},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that save can override just the transform
func TestSaveThenOverrideTransform(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_file_transform")
	defer run.Delete()

	// TODO(dlong): Move into TestRunner, use this everywhere.
	prevXformVer := startf.Version
	startf.Version = "test_version"
	defer func() {
		startf.Version = prevXformVer
	}()

	// Save a version, then save another with a transform
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/tf.star me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	actual := run.DatasetMarshalJSON(t, dsPath)

	// This dataset is ds_ten.yaml, with an added transform section
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"transform added","path":"/ipfs/QmWq1C8kx6d4Fe5hBsUaqXFh2VDUMzaE5ewTNjqgGnXivV","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"transform added"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/QmTPdKe6Erfft7MEfcLufYCn9yBPYCQTvLfViop14rGUGa","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/Qmb69tx5VCL7q7EfkGKpDgESBysmDbohoLvonpbgri48NN","syntax":"starlark","syntaxVersion":"test_version"},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmXkN5J5yCAtF8GCxwRXARzAQhj3bPaSv1VHoyCCXzQRzN","scriptPath":"/ipfs/QmVM37PFzBcZn3qqKvyQ9rJ1jC8NkS8kYZNJke1Wje1jor"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that save can override just the viz
func TestSaveThenOverrideViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_file_transform")
	defer run.Delete()

	// Save a version, then save another with a viz template
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/template.html me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	actual := run.DatasetMarshalJSON(t, dsPath)

	// This dataset is ds_ten.yaml, with an added viz section
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"viz:\n\tupdated scriptPath","path":"/ipfs/QmUqJPrAUPQHTbnoYvbcuHDYpLLa3cLJucfPS5gDYP7YPF","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"viz updated scriptPath"},"meta":{"qri":"md:0","title":"example movie data"},"path":"/ipfs/QmNaJ1Wi774Ghh1fviRz1GNQ2UFAcCb96z4kTPszZkKAGt","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmVrEH7T7XmdJLym8YL9DjwCALbz264h7GQTrjkSGmbvry","scriptPath":"/ipfs/QmRaVGip3V9fVBJheZN6FbUajD3ZLNjHhXdjrmfg2JPoo5"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that save can combine a meta compoent, and a transform, and a viz
func TestSaveThenOverrideMetaAndTransformAndViz(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_file_transform")
	defer run.Delete()

	// TODO(dlong): Move into TestRunner, use this everywhere.
	prevXformVer := startf.Version
	startf.Version = "test_version"
	defer func() {
		startf.Version = prevXformVer
	}()

	// Save a version, then save another with three components at once
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/tf.star --file=testdata/template.html me/test_ds")

	// Read head from the dataset that was saved, as json string.
	dsPath := run.GetPathForDataset(t, 0)
	actual := run.DatasetMarshalJSON(t, dsPath)

	// This dataset is ds_ten.yaml, with an added meta component, and transform, and viz
	expect := `{"bodyPath":"/ipfs/QmXhsUK6vGZrqarhw9Z8RCXqhmEpvtVByKtaYVarbDZ5zn","commit":{"author":{"id":"QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B"},"message":"meta:\n\tupdated title\nviz:\n\tupdated scriptPath\ntransform added","path":"/ipfs/QmPtYLgM886MQCQhsyJDgb4FnNqNH8FBBDE1NqwJpJi4Js","qri":"cm:0","signature":"njCFxpGqq0xJSrjgxC289KncjflqA0e00txweEqIyUTvEKSUBKHcfQmx4OQIJzJqQJdcjIEzFrwP9cdquozRgsnrpsSfKb+wBWdtbnrg8zfat0X/Dqjro6JD7afJf0gU9s5SDi/s8g/qZOLwWh1nuoH4UAeUX+l3DH0ocFjeD6r/YkMJ0KXaWaFloKP8UPasfqoei9PxxmYQuAnFMqpXFisB7mKFAbgbpF3eL80UcbQPTih7WF11SBym/AzJhGNvOivOjmRxKGEuqEH9g3NPTEQr+LnP415X4qiaZA6MVmOO66vC0diUN4vJUMvhTsWnVEBtgqjTRYlSaYwabHv/gA==","timestamp":"2001-01-01T01:02:01.000000001Z","title":"updated meta, viz, and transform"},"meta":{"qri":"md:0","title":"different title"},"path":"/ipfs/QmSJyjtKNNBoEJGNAXLStagJbziiaBskDbrxJPesg2JLW1","peername":"me","previousPath":"/ipfs/QmVdDACqmUoFGCotChqSuYJMnocPwkXPifEB6kGqiTjhiL","qri":"ds:0","structure":{"checksum":"QmcXDEGeWdyzfFRYyPsQVab5qszZfKqxTMEoXRDSZMyrhf","depth":2,"errCount":1,"entries":8,"format":"csv","formatConfig":{"headerRow":true,"lazyQuotes":true},"length":224,"qri":"st:0","schema":{"items":{"items":[{"title":"movie_title","type":"string"},{"title":"duration","type":"integer"}],"type":"array"},"type":"array"}},"transform":{"qri":"tf:0","scriptPath":"/ipfs/Qmb69tx5VCL7q7EfkGKpDgESBysmDbohoLvonpbgri48NN","syntax":"starlark","syntaxVersion":"test_version"},"viz":{"format":"html","qri":"vz:0","renderedPath":"/ipfs/QmVrEH7T7XmdJLym8YL9DjwCALbz264h7GQTrjkSGmbvry","scriptPath":"/ipfs/QmRaVGip3V9fVBJheZN6FbUajD3ZLNjHhXdjrmfg2JPoo5"}}`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("dataset (-want +got):\n%s", diff)
	}
}

// Test that saving a full dataset with a component at the same time is an error
func TestSaveDatasetWithComponentError(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Try to save with two conflicting components, but this returns an error
	err := run.ExecCommand("qri save --file=testdata/movies/ds_ten.yaml --file=testdata/movies/meta_override.yaml me/test_ds")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}
	expect := `conflict, cannot save a full dataset with other components`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that saving with two components of the same kind is an error
func TestSaveConflictingComponents(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_then_override_meta")
	defer run.Delete()

	// Save two versions, but second has a conflict error
	run.MustExec(t, "qri save --file=testdata/movies/ds_ten.yaml me/test_ds")
	err := run.ExecCommand("qri save --file=testdata/movies/meta_override.yaml --file=testdata/movies/meta_override.yaml me/test_ds")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}
	expect := `conflict, multiple components of kind "md"`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that running a transform without any changes will not make a new commit
func TestSaveTransformWithoutChanges(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_transform_same")
	defer run.Delete()

	// Save a version, then another with no changes
	run.MustExec(t, "qri save --file=testdata/movies/tf_123.star me/test_ds")
	err := run.ExecCommand("qri save --file=testdata/movies/tf_123.star me/test_ds")
	expect := `error saving: no changes`
	if err == nil {
		t.Fatalf("expected error: did not get one")
	}
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that calling `get_body` will retrieve the body of the previous version.
func TestTransformUsingGetBodyAndSetBody(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_save_transform_get_body")
	defer run.Delete()

	// Save two versions, the second of which uses get_body in a transformation
	run.MustExec(t, "qri save --body=testdata/movies/body_two.json me/test_ds")
	run.MustExec(t, "qri save --file=testdata/movies/tf_add_one.star me/test_ds")

	// Read body from the dataset that was created with the transform
	dsPath := run.GetPathForDataset(t, 0)
	actualBody := run.ReadBodyFromIPFS(t, dsPath+"/body.json")

	// This body is body_two.json, with the numbers in the second column increased by 1.
	expectBody := `[["Avatar",179],["Pirates of the Caribbean: At World's End",170]]`
	if actualBody != expectBody {
		t.Errorf("error, dataset actual:\n%s\nexpect:\n%s\n", actualBody, expectBody)
	}
}

// Test that we can compare bodies of different dataset revisions.
func TestDiffRevisions(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_diff_revisions")
	defer run.Delete()

	// Save three versions, then diff the last two
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_twenty.csv me/test_movies")
	run.MustExec(t, "qri save --body=testdata/movies/body_thirty.csv me/test_movies")
	output := run.MustExec(t, "qri diff body me/test_movies")

	expect := `+30 elements. 10 inserts. 0 deletes.

 0: ["Avatar ",178]
 1: ["Pirates of the Caribbean: At World's End ",169]
 2: ["Spectre ",148]
 3: ["The Dark Knight Rises ",164]
 4: ["Star Wars: Episode VII - The Force Awakens             ",""]
 5: ["John Carter ",132]
 6: ["Spider-Man 3 ",156]
 7: ["Tangled ",100]
 8: ["Avengers: Age of Ultron ",141]
 9: ["Harry Potter and the Half-Blood Prince ",153]
 10: ["Batman v Superman: Dawn of Justice ",183]
 11: ["Superman Returns ",169]
 12: ["Quantum of Solace ",106]
 13: ["Pirates of the Caribbean: Dead Man's Chest ",151]
 14: ["The Lone Ranger ",150]
 15: ["Man of Steel ",143]
 16: ["The Chronicles of Narnia: Prince Caspian ",150]
 17: ["The Avengers ",173]
+18: ["Dragonfly ",104]
+19: ["The Black Dahlia ",121]
+20: ["Flyboys ",140]
+21: ["The Last Castle ",131]
+22: ["Supernova ",91]
+23: ["Winter's Tale ",118]
+24: ["The Mortal Instruments: City of Bones ",130]
+25: ["Meet Dave ",90]
+26: ["Dark Water ",103]
+27: ["Edtv ",122]
`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// Test that diffing a dataset with only one version produces an error
func TestDiffOnlyOneRevision(t *testing.T) {
	if err := confirmQriNotRunning(); err != nil {
		t.Skip(err.Error())
	}

	run := NewTestRunner(t, "test_peer", "qri_test_diff_only_one")
	defer run.Delete()

	// Save a version, then try to diff but it returns an error because there's only one version
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/test_movies")
	err := run.ExecCommand("qri diff body me/test_movies")
	if err == nil {
		t.Errorf("expected error, did not get one")
	}
	expect := `dataset has only one version, nothing to diff against`
	if err.Error() != expect {
		t.Errorf("expected error: \"%s\", got: \"%s\"", expect, err.Error())
	}
}

// Test that save can be called with a readme file
func TestSaveReadmeFromFile(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "save_readme_file")
	defer run.Delete()

	// Save two versions, one with a body, the second with a readme
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/save_readme_file")
	run.MustExec(t, "qri save --file=testdata/movies/about_movies.md me/save_readme_file")

	// Verify we can get the readme back
	actual := run.MustExec(t, "qri get readme me/save_readme_file")
	expect := `format: md
qri: rm:0
scriptPath: /ipfs/QmQPbLdDwyAzCmKayuHGeNGx5eboDv5aXTMuw2daUuneCb

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("readme.md contents (-want +got):\n%s", diff)
	}

	// As well as the readme script bytes
	actual = run.MustExec(t, "qri get readme.script me/save_readme_file")
	expect = `# Title

This is a dataset about movies

`
	if diff := cmp.Diff(expect, actual); diff != "" {
		t.Errorf("readme.md contents (-want +got):\n%s", diff)
	}
}

// Test that renaming a dataset after registration (which changes the username) works correctly
func TestRenameAfterRegistration(t *testing.T) {
	run := NewTestRunnerWithTempRegistry(t, "test_peer", "rename_after_reg")
	defer run.Delete()

	// Create a dataset, using the "anonymous" generated username.
	run.MustExec(t, "qri save --body=testdata/movies/body_ten.csv me/first_name")

	// Verify the raw references in the repo
	output := run.MustExec(t, "qri list --raw")
	expect := `0 Peername:  test_peer
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      first_name
  Path:      /ipfs/QmezSToWr8oAvqFbMGFrMuKj5srNK5YPsN4tHUw2JGnWra
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Register (using a mock server) which changes the username
	run.MustExec(t, "qri registry signup --username real_peer --email me@example.com --password hi")

	output = run.MustExec(t, "qri list --raw")
	expect = `0 Peername:  real_peer
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      first_name
  Path:      /ipfs/QmezSToWr8oAvqFbMGFrMuKj5srNK5YPsN4tHUw2JGnWra
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Rename the created dataset, which should work even though our username changed
	run.MustExec(t, "qri rename me/first_name me/second_name")

	output = run.MustExec(t, "qri list --raw")
	expect = `0 Peername:  real_peer
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      second_name
  Path:      /ipfs/QmezSToWr8oAvqFbMGFrMuKj5srNK5YPsN4tHUw2JGnWra
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

	// Rename a second time, make sure this works still
	run.MustExec(t, "qri rename me/second_name me/third_name")

	output = run.MustExec(t, "qri list --raw")
	expect = `0 Peername:  real_peer
  ProfileID: QmeL2mdVka1eahKENjehK6tBxkkpk5dNQ1qMcgWi7Hrb4B
  Name:      third_name
  Path:      /ipfs/QmezSToWr8oAvqFbMGFrMuKj5srNK5YPsN4tHUw2JGnWra
  FSIPath:   
  Published: false

`
	if diff := cmp.Diff(expect, output); diff != "" {
		t.Errorf("unexpected (-want +got):\n%s", diff)
	}

}
