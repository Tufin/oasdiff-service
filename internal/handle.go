package internal

import (
	"io"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	log "github.com/sirupsen/logrus"
	"github.com/tufin/oasdiff/diff"
	"github.com/tufin/oasdiff/load"
	"gopkg.in/yaml.v3"
)

func Diff(w http.ResponseWriter, r *http.Request) {

	dir, base, revision, code := createFiles(r)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}
	defer closeFile(base)
	defer closeFile(revision)
	defer os.RemoveAll(dir)

	diffReport, code := createDiffReport(base, revision)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err := yaml.NewEncoder(w).Encode(diffReport)
	if err != nil {
		log.Errorf("failed to encode diff report with %v", err)
	}
}

func createDiffReport(base *os.File, revision *os.File) (*diff.Diff, int) {

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	s1, err := loader.LoadFromFile(base.Name())
	if err != nil {
		log.Infof("failed to load base spec from %q with %q", base.Name(), err)
		return nil, http.StatusBadRequest
	}

	s2, err := load.From(loader, revision.Name())
	if err != nil {
		log.Infof("failed to load revision spec from %q with %q", revision.Name(), err)
		return nil, http.StatusBadRequest
	}

	config := diff.NewConfig()
	config.ExcludeExamples = false
	config.ExcludeDescription = false
	config.PathFilter = ""
	config.FilterExtension = ""
	config.PathPrefixBase = ""
	config.PathPrefixRevision = ""
	config.PathStripPrefixBase = ""
	config.PathStripPrefixRevision = ""
	config.BreakingOnly = false
	config.DeprecationDays = 0

	diffReport, err := diff.Get(config, s1, s2)
	if err != nil {
		log.Infof("failed to load revision spec from %q with %q", revision.Name(), err)
		return nil, http.StatusBadRequest
	}

	return diffReport, http.StatusOK
}

func createFiles(r *http.Request) (string, *os.File, *os.File, int) {

	// 32 MB is the default used by FormFile() function
	if err := r.ParseMultipartForm(4); err != nil {
		log.Errorf("failed to parse HTTP request files with %v", err)
		return "", nil, nil, http.StatusInternalServerError
	}

	// create a temporary directory
	dir, err := os.MkdirTemp("", "tmp")
	if err != nil {
		log.Errorf("failed to make temp dir with %v", err)
		return "", nil, nil, http.StatusInternalServerError
	}

	base, code := createFile(r, dir, "base")
	if code != http.StatusOK {
		os.RemoveAll(dir)
		return "", nil, nil, code
	}
	revision, code := createFile(r, dir, "revision")
	if code != http.StatusOK {
		os.RemoveAll(dir)
		closeFile(base)
		return "", nil, nil, code
	}

	return dir, base, revision, http.StatusOK
}

func createFile(r *http.Request, dir string, filename string) (*os.File, int) {

	// create and open a temporary file
	res, err := os.CreateTemp(dir, "")
	if err != nil {
		log.Errorf("failed to create temp directory with %v", err)
		return nil, http.StatusInternalServerError
	}

	// a reference to the fileHeaders are accessible only after ParseMultipartForm is called
	files := r.MultipartForm.File[filename]
	for _, fileHeader := range files {
		// Open the file
		file, err := fileHeader.Open()
		if err != nil {
			log.Errorf("failed to create temp file with %v", err)
			return nil, http.StatusInternalServerError
		}
		defer file.Close()

		_, err = io.Copy(res, file)
		if err != nil {
			log.Errorf("failed to copy file %q from HTTP request with %v", fileHeader.Filename, err)
			return nil, http.StatusInternalServerError
		}
	}

	return res, http.StatusOK
}

func closeFile(f *os.File) {

	err := f.Close()
	if err != nil {
		log.Errorf("failed to close 'revision' file with %v", err)
	}
}
