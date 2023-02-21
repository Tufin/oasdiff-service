package internal

import (
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	log "github.com/sirupsen/logrus"
	"github.com/tufin/oasdiff/diff"
	"github.com/tufin/oasdiff/load"
	"gopkg.in/yaml.v3"
)

func Diff(w http.ResponseWriter, r *http.Request) {

	dir, base, revision, code := CreateFiles(r)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}
	defer CloseFile(base)
	defer CloseFile(revision)
	defer os.RemoveAll(dir)

	diffReport, code := createDiffReport(base, revision)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/yaml")
	err := yaml.NewEncoder(w).Encode(diffReport)
	if err != nil {
		log.Errorf("failed to encode diff report with %v", err)
	}
}

func createDiffReport(base *os.File, revision *os.File) (*diff.Diff, int) {

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	s1, err := loader.LoadFromFile(base.Name())
	if err != nil {
		log.Infof("failed to load base spec from %q with %v", base.Name(), err)
		return nil, http.StatusBadRequest
	}
	s2, err := load.From(loader, revision.Name())
	if err != nil {
		log.Infof("failed to load revision spec from %q with %v", revision.Name(), err)
		return nil, http.StatusBadRequest
	}
	config := CreateConfig()

	diffReport, err := diff.Get(config, s1, s2)
	if err != nil {
		log.Infof("failed to load revision spec from %q with %v", revision.Name(), err)
		return nil, http.StatusBadRequest
	}

	return diffReport, http.StatusOK
}
