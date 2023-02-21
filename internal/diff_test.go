package internal_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tufin/oasdiff-service/internal"
	"github.com/tufin/oasdiff/diff"
	"gopkg.in/yaml.v3"
)

func TestDiff(t *testing.T) {

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	basePart, err := writer.CreateFormFile("base", "openapi-test1.yaml")
	require.NoError(t, err)
	base, err := os.Open("../data/openapi-test1.yaml")
	require.NoError(t, err)
	_, err = io.Copy(basePart, base)
	require.NoError(t, err)

	revisionPart, err := writer.CreateFormFile("revision", "openapi-test3.yaml")
	require.NoError(t, err)
	revision, err := os.Open("../data/openapi-test3.yaml")
	require.NoError(t, err)
	_, err = io.Copy(revisionPart, revision)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	r, err := http.NewRequest(http.MethodPost, "/diff", body)
	require.NoError(t, err)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	internal.Diff(w, r)

	require.Equal(t, http.StatusCreated, w.Result().StatusCode)
	var report diff.Diff
	yaml.NewDecoder(w.Result().Body).Decode(&report)
	require.NoError(t, err)
	require.False(t, report.Empty())
}
