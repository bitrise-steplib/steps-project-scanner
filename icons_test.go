package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bitrise-io/bitrise-init/models"
)

func Test_uploadIcons(t *testing.T) {
	const testFileIDQueryKey = "testfileid"
	fileIDtoSize := make(map[string]int64)
	var iconCandidates []models.Icon

	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("httptest: failed to read request, error: %s", err)
			return
		}

		testFileID := r.URL.Query().Get(testFileIDQueryKey)
		if testFileID == "" {
			t.Errorf("httptest: no test file id specified")
			return
		}

		wantFileSize, found := fileIDtoSize[testFileID]
		if !found {
			t.Errorf("httptest: test file id not found")
		}

		if int64(len(bytes)) != wantFileSize {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("httptest: failed to read body: %s", err)
		}

		candidates := []appIconCandidateURL{}
		if err := json.Unmarshal(bytes, &candidates); err != nil {
			t.Errorf("httptest: failed to unmarshal request, error: %s", err)
		}

		var responseCandidates []appIconCandidateURL
		for _, candidate := range candidates {
			fileIDtoSize[candidate.FileName] = candidate.FileSize
			responseCandidates = append(responseCandidates, appIconCandidateURL{
				FileName:  candidate.FileName,
				FileSize:  candidate.FileSize,
				UploadURL: storage.URL + fmt.Sprintf("?%s=%s", testFileIDQueryKey, url.QueryEscape(candidate.FileName)),
			})
		}

		response := map[string][]appIconCandidateURL{
			"data": responseCandidates,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Errorf("httptest: failed to marshal response, error: %s", err)
		}

		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(responseBytes); err != nil {
			t.Error("httptest: failed to write response")
		}
	}))

	for i := 0; i < 10; i++ {
		file, err := os.CreateTemp("", "")
		if err != nil {
			t.Errorf("setup: failed to create file, error: %s", err)
		}
		pathAbs, err := filepath.Abs(file.Name())
		if err != nil {
			t.Errorf("setup: failed to get file path, error: %s", err)
		}

		img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{rand.Intn(100) + 1, rand.Intn(100) + 1}})
		if err := png.Encode(file, img); err != nil {
			t.Errorf("setup: failed to write file, error: %s", err)
		}

		fileID := strconv.Itoa(rand.Int()) + ".png"
		iconCandidates = append(iconCandidates, models.Icon{
			Filename: fileID,
			Path:     pathAbs,
		})

		err = file.Close()
		if err != nil {
			t.Errorf("setup: failed to close file")
		}
	}

	type args struct {
		icons []models.Icon
		query iconCandidateQuery
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy case",
			args: args{
				icons: iconCandidates,
				query: iconCandidateQuery{
					URL:               api.URL,
					buildTriggerToken: "token",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := uploadIcons(tt.args.icons, tt.args.query); (err != nil) != tt.wantErr {
				t.Errorf("uploadIcons() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
