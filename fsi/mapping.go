package fsi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
	"github.com/qri-io/dataset"
)

const (
	componentNameCommit    = "commit"
	componentNameDataset   = "dataset"
	componentNameMeta      = "meta"
	componentNameSchema    = "schema"
	componentNameBody      = "body"
	componentNameStructure = "structure"
	componentNameTransform = "transform"
	componentNameViz       = "viz"
)

var (
	// ErrNoDatasetFiles indicates no data
	ErrNoDatasetFiles = fmt.Errorf("no dataset files provided")
)

// FileStat holds information about a file: its path and mod time
type FileStat struct {
	Path  string
	Mtime time.Time
}

// ReadDir parses a directory into a dataset, returning both the dataset and
// a map of component names to the files they came from. Files can be specified
// in either JSON or YAML format. It is an error to specify any component more
// than once
func ReadDir(dir string) (ds *dataset.Dataset, fileMap, problems map[string]FileStat, err error) {
	fileMap = map[string]FileStat{}
	ds = &dataset.Dataset{}
	schema := map[string]interface{}{}
	problems = nil

	components := map[string]interface{}{
		componentNameDataset: ds,

		componentNameCommit:    &dataset.Commit{},
		componentNameMeta:      &dataset.Meta{},
		componentNameStructure: &dataset.Structure{},
		componentNameSchema:    &schema,
		componentNameTransform: &dataset.Transform{},
		componentNameViz:       &dataset.Viz{},

		componentNameBody: nil,
	}

	extensions := map[string]decoderFactory{
		".json": newJSONDecoder,
		".yaml": newYAMLDecoder,
		".yml":  newYAMLDecoder,
	}

	addFile := func(cmpName, path string, mtime time.Time) error {
		foundFile, exists := fileMap[cmpName]
		if exists {
			foundPath := filepath.Base(foundFile.Path)
			anotherPath := filepath.Base(path)
			return fmt.Errorf(`%s is defined in two places: %s and %s. please remove one`, cmpName, foundPath, anotherPath)
		}
		fileMap[cmpName] = FileStat{Path: path, Mtime: mtime}
		return nil
	}

	// HACK: Detect body format and body modification time.
	var mtime time.Time
	bodyFormat := ""
	if st, err := os.Stat(filepath.Join(dir, "body.csv")); !os.IsNotExist(err) {
		mtime = st.ModTime()
		bodyFormat = "csv"
	}
	if st, err := os.Stat(filepath.Join(dir, "body.json")); !os.IsNotExist(err) {
		if bodyFormat == "csv" {
			return ds, fileMap, problems, fmt.Errorf("body.csv and body.json both exist")
		}
		mtime = st.ModTime()
		bodyFormat = "json"
	}

	bodyFilename := ""
	if bodyFormat != "" {
		bodyFilename = fmt.Sprintf("body.%s", bodyFormat)
		if err = addFile(componentNameBody, bodyFilename, mtime); err != nil {
			return ds, fileMap, problems, err
		}
		if ds.BodyPath == "" {
			ds.BodyPath = filepath.Join(dir, bodyFilename)
		}
	}

	var st os.FileInfo

	// Iterate components in a deterministic order, from highest priority to lowest.
	for i := 0; i < len(componentListOrder); i++ {
		cmpName := componentListOrder[i]
		cmp := components[cmpName]
		for ext, mkDec := range extensions {
			filename := fmt.Sprintf("%s%s", cmpName, ext)
			path := filepath.Join(dir, filename)
			if f, e := os.Open(path); e == nil {
				st, _ = f.Stat()
				if cmpName != componentNameBody {
					if err = mkDec(f).Decode(cmp); err != nil {
						if problems == nil {
							problems = make(map[string]FileStat)
						}
						problems[cmpName] = FileStat{Path: filename, Mtime: st.ModTime()}
						// Don't treat this parse failure as an error, only as a "problem" to
						// display in `status`. This prevents the entire function from returning
						// an error in the case that no other components are checked after this
						// one.
						err = nil
						continue
					}
					if err = addFile(cmpName, path, st.ModTime()); err != nil {
						return ds, fileMap, problems, err
					}
				}

				switch cmpName {
				case componentNameDataset:
					if ds.Commit != nil {
						if err = addFile(componentNameCommit, path, st.ModTime()); err != nil {
							return
						}
					}
					if ds.Meta != nil {
						if err = addFile(componentNameMeta, path, st.ModTime()); err != nil {
							return
						}
					}
					if ds.Structure != nil {
						if err = addFile(componentNameStructure, path, st.ModTime()); err != nil {
							return
						}
						if ds.Structure.Schema != nil {
							if err = addFile(componentNameSchema, path, st.ModTime()); err != nil {
								return
							}
						}
					}
					if ds.Viz != nil {
						if err = addFile(componentNameViz, path, st.ModTime()); err != nil {
							return
						}
					}
					if ds.Transform != nil {
						if err = addFile(componentNameTransform, path, st.ModTime()); err != nil {
							return
						}
					}
					if ds.Body != nil {
						if err = addFile(componentNameBody, path, st.ModTime()); err != nil {
							return
						}
					}

				case componentNameCommit:
					ds.Commit = cmp.(*dataset.Commit)
				case componentNameMeta:
					ds.Meta = cmp.(*dataset.Meta)
				case componentNameStructure:
					ds.Structure = cmp.(*dataset.Structure)
				case componentNameSchema:
					if ds.Structure == nil {
						ds.Structure = &dataset.Structure{}
					}
					ds.Structure.Schema = *cmp.(*map[string]interface{})
					if ds.Structure.Format == "" {
						ds.Structure.Format = bodyFormat
					}
				case componentNameViz:
					ds.Viz = cmp.(*dataset.Viz)
				case componentNameTransform:
					ds.Transform = cmp.(*dataset.Transform)
				case componentNameBody:
					if ds.BodyPath == "" {
						ds.BodyPath = path
					}
				}
			}
		}
	}

	if len(fileMap) == 0 {
		err = ErrNoDatasetFiles
	}

	return ds, fileMap, problems, err
}

type decoderFactory func(io.Reader) decoder

type decoder interface {
	Decode(m interface{}) error
}

type jsonDecoder struct {
	dec *json.Decoder
}

func newJSONDecoder(r io.Reader) decoder {
	return jsonDecoder{
		dec: json.NewDecoder(r),
	}
}

func (jd jsonDecoder) Decode(v interface{}) error {
	return jd.dec.Decode(v)
}

type yamlDecoder struct {
	rdr io.Reader
}

func newYAMLDecoder(r io.Reader) decoder {
	return yamlDecoder{
		rdr: r,
	}
}

func (yd yamlDecoder) Decode(v interface{}) error {
	// convert yaml input to json as a hack to support yaml input for now
	yamlData, err := ioutil.ReadAll(yd.rdr)
	if err != nil {
		return fmt.Errorf("invalid file: %s", err.Error())
	}

	jsonData, err := yaml.YAMLToJSON(yamlData)
	if err != nil {
		return fmt.Errorf("converting yaml body to json: %s", err.Error())
	}

	return json.Unmarshal(jsonData, v)
}

// WriteComponents writes components of the dataset to the given path, as individual files.
func WriteComponents(ds *dataset.Dataset, dirPath string) error {
	// Get individual meta and schema components.
	meta := ds.Meta
	ds.Meta = nil

	var bodyFormat string
	var schema map[string]interface{}
	if ds.Structure != nil {
		schema = ds.Structure.Schema
		ds.Structure.Schema = nil

		bodyFormat = ds.Structure.Format

		// Structure is kept in the dataset.
		ds.Structure.Format = ""
		ds.Structure.Qri = ""
	}

	// Commit, viz, transform are never written as individual files.
	ds.Commit = nil
	ds.Viz = nil
	ds.Transform = nil

	// Meta component.
	if meta != nil {
		meta.DropDerivedValues()
		if !meta.IsEmpty() {
			data, err := json.MarshalIndent(meta, "", " ")
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(dirPath, "meta.json"), data, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	// Schema component.
	if len(schema) > 0 {
		data, err := json.MarshalIndent(schema, "", " ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dirPath, "schema.json"), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Body component.
	bf := ds.BodyFile()
	if bf != nil {
		data, err := ioutil.ReadAll(bf)
		if err != nil {
			return err
		}
		ds.BodyPath = ""
		var bodyFilename string
		switch bodyFormat {
		case "csv":
			bodyFilename = "body.csv"
		case "json":
			bodyFilename = "body.json"
		default:
			return fmt.Errorf(`unknown body format: "%s"`, bodyFormat)
		}
		err = ioutil.WriteFile(filepath.Join(dirPath, bodyFilename), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Dataset (everything else).
	ds.DropDerivedValues()
	// TODO(dlong): Should more of these move to DropDerivedValues?
	ds.Qri = ""
	ds.Name = ""
	ds.Peername = ""
	ds.PreviousPath = ""
	if ds.Structure != nil && ds.Structure.IsEmpty() {
		ds.Structure = nil
	}
	if !ds.IsEmpty() {
		data, err := json.MarshalIndent(ds, "", " ")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dirPath, "dataset.json"), data, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteComponents removes the list of named components from the given directory
func DeleteComponents(removeList []string, fileMap map[string]FileStat, dirPath string) error {
	for _, comp := range removeList {
		removeFile := fileMap[comp].Path
		// TODO(dlong): Collect errors and return them all, instead of bailing immediately
		if err := os.Remove(removeFile); err != nil {
			return err
		}
	}
	return nil
}