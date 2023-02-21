package internal

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/tufin/oasdiff/checker"
	"github.com/tufin/oasdiff/checker/localizations"
	"github.com/tufin/oasdiff/diff"
	"gopkg.in/yaml.v3"
)

func BreakingChanges(w http.ResponseWriter, r *http.Request) {

	dir, base, revision, code := CreateFiles(r)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}
	defer CloseFile(base)
	defer CloseFile(revision)
	defer os.RemoveAll(dir)

	breakingChanges, code := calcBreakingChanges(r, base.Name(), revision.Name())
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/yaml")
	err := yaml.NewEncoder(w).Encode(map[string][]checker.BackwardCompatibilityError{"breaking-changes": breakingChanges})
	if err != nil {
		log.Errorf("failed to encode diff report with %v", err)
	}
}

func calcBreakingChanges(r *http.Request, base string, revision string) ([]checker.BackwardCompatibilityError, int) {

	config := CreateConfig()

	// breaking changes
	config.IncludeExtensions.Add(checker.XStabilityLevelExtension)
	config.IncludeExtensions.Add(diff.SunsetExtension)
	config.IncludeExtensions.Add(checker.XExtensibleEnumExtension)

	s1, err := checker.LoadOpenAPISpecInfo(base)
	if err != nil {
		log.Infof("failed to load base spec from %q with %v", base, err)
		return nil, http.StatusBadRequest
	}
	s2, err := checker.LoadOpenAPISpecInfo(revision)
	if err != nil {
		log.Infof("failed to load revision spec from %q with %v", revision, err)
		return nil, http.StatusBadRequest
	}
	diffReport, operationsSources, err := diff.GetWithOperationsSourcesMap(config, s1, s2)
	if err != nil {
		log.Errorf("failed to 'diff.GetWithOperationsSourcesMap' with %v", err)
		return nil, http.StatusInternalServerError
	}

	c := checker.DefaultChecks()
	c.Localizer = *localizations.New(getLocal(r), "en")

	return checker.CheckBackwardCompatibility(c, diffReport, operationsSources), http.StatusOK
}

func getLocal(r *http.Request) string {

	local := r.URL.Query()["local"]
	if local != nil && local[0] != "" {
		return local[0]
	}

	return "en"
}
