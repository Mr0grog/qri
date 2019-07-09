package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
	"github.com/qri-io/ioes"
	"github.com/qri-io/qri/lib"
	"github.com/spf13/cobra"
)

// NewInitCommand creates new `qri init` command that connects a working directory in
// the local filesystem to a dataset your repo.
func NewInitCommand(f Factory, ioStreams ioes.IOStreams) *cobra.Command {
	o := &InitOptions{IOStreams: ioStreams}
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "initialize a dataset directory",
		Long:    ``,
		Example: ``,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Complete(f)
			return o.Run()
		},
	}

	cmd.PersistentFlags().StringVar(&o.Name, "name", "", "name of the dataset")
	cmd.PersistentFlags().StringVar(&o.Format, "format", "", "format of dataset")

	return cmd
}

// InitOptions encapsulates state for the `init` command
type InitOptions struct {
	ioes.IOStreams

	Name            string
	Format          string
	DatasetRequests *lib.DatasetRequests
}

// Complete completes a dataset reference
func (o *InitOptions) Complete(f Factory) (err error) {
	o.DatasetRequests, err = f.DatasetRequests()
	return err
}

// Run executes the `init` command
func (o *InitOptions) Run() (err error) {
	if _, err := os.Stat(QriRefFilename); !os.IsNotExist(err) {
		return fmt.Errorf("working directory is already linked, .qri-ref exists")
	}
	if _, err := os.Stat("meta.json"); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the meta.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, meta.json exists")
	}
	if _, err := os.Stat("schema.json"); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the schema.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, schema.json exists")
	}
	if _, err := os.Stat("body.csv"); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the body.csv file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, body.csv exists")
	}
	if _, err := os.Stat("body.json"); !os.IsNotExist(err) {
		// TODO(dlong): Instead, import the body.json file for the new dataset
		return fmt.Errorf("cannot initialize new dataset, body.json exists")
	}

	// Suggestion for the dataset name defaults to the name of the current directory.
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	suggestDataset := filepath.Base(pwd)

	// Process flags for inputs, prompt for any that were not provided.
	var dsName, dsFormat string
	if o.Name != "" {
		dsName = o.Name
	} else {
		dsName = inputText(o.ErrOut, o.In, "Name of new dataset", suggestDataset)
	}
	if o.Format != "" {
		dsFormat = o.Format
	} else {
		dsFormat = inputText(o.ErrOut, o.In, "Format of dataset, csv or json", "csv")
	}

	ref := fmt.Sprintf("me/%s", dsName)

	// Validate dataset name. The `init` command must only be used for creating new datasets.
	// Make sure a dataset with this name does not exist in your repo.
	p := lib.GetParams{
		Path:     ref,
		Selector: "",
	}
	res := lib.GetResult{}
	if err = o.DatasetRequests.Get(&p, &res); err == nil {
		// TODO(dlong): Tell user to use `checkout` if the dataset already exists in their repo?
		return fmt.Errorf("a dataset with the name %s already exists in your repo", ref)
	}

	// Validate dataset format
	if dsFormat != "csv" && dsFormat != "json" {
		return fmt.Errorf("invalid format \"%s\", only \"csv\" and \"json\" accepted", dsFormat)
	}

	// Create the link file, containing the dataset reference.
	if err = ioutil.WriteFile(QriRefFilename, []byte(ref), os.ModePerm); err != nil {
		return fmt.Errorf("creating %s file: %s", QriRefFilename, err)
	}

	// Create a skeleton meta.json file.
	meta := dataset.Meta{
		Qri:         "md:0",
		Citations:   []*dataset.Citation{},
		Description: "enter description here",
		Title:       "enter title here",
		HomeURL:     "enter home URL here",
		Keywords:    []string{"example"},
	}
	data, err := json.MarshalIndent(meta, "", " ")
	if err := ioutil.WriteFile("meta.json", data, os.ModePerm); err != nil {
		return err
	}

	var schema map[string]interface{}
	if dsFormat == "csv" {
		schema = map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "array",
				"items": []interface{}{
					// First column
					map[string]interface{}{
						"type":  "string",
						"title": "name",
					},
					// Second column
					map[string]interface{}{
						"type":  "string",
						"title": "describe",
					},
					// Third column
					map[string]interface{}{
						"type":  "integer",
						"title": "quantity",
					},
				},
			},
		}
	} else {
		schema = map[string]interface{}{
			"type": "object",
		}
	}
	data, err = json.MarshalIndent(schema, "", " ")
	if err := ioutil.WriteFile("schema.json", data, os.ModePerm); err != nil {
		return err
	}

	// Create a skeleton body file.
	if dsFormat == "csv" {
		bodyText := "one,two,3\nfour,five,6"
		if err := ioutil.WriteFile("body.csv", []byte(bodyText), os.ModePerm); err != nil {
			return err
		}
	} else {
		bodyText := `{
  "key": "value"
}`
		if err := ioutil.WriteFile("body.json", []byte(bodyText), os.ModePerm); err != nil {
			return err
		}
	}

	printSuccess(o.Out, "initialized working directory for new dataset %s", ref)
	return nil
}
