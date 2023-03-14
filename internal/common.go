package internal

import (
	"io"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/tufin/oasdiff/diff"
)

func CreateConfig() *diff.Config {

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

	return config
}

func CreateFiles(r *http.Request) (string, *os.File, *os.File, int) {

	contentType := r.Header.Get("Content-Type")
	if contentType == "multipart/form-data" {
		// 32 MB is the default used by FormFile() function
		if err := r.ParseMultipartForm(4); err != nil {
			log.Errorf("failed to parse 'multipart/form-data' request files with '%v'", err)
			return "", nil, nil, http.StatusBadRequest
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("failed to parse form request with '%v'", err)
			return "", nil, nil, http.StatusBadRequest
		}
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
		CloseFile(base)
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

func CloseFile(f *os.File) {

	err := f.Close()
	if err != nil {
		log.Errorf("failed to close file with %v", err)
	}
}
