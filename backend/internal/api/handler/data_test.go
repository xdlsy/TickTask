package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"ticktask/internal/model"
	"ticktask/internal/service"
)

var errBoom = errors.New("boom")

// mockDataService 实现 service.DataService
type mockDataService struct {
	exportEnvelop    *model.BackupEnvelope
	exportErr        error
	previewResult    *model.ImportPreview
	previewErr       error
	applyResult      *model.ApplyResult
	applyErr         error
	lastFileVersion  int
	lastIncludeKey   bool
}

func (m *mockDataService) Export(includeAPIKey bool) (*model.BackupEnvelope, error) {
	m.lastIncludeKey = includeAPIKey
	return m.exportEnvelop, m.exportErr
}
func (m *mockDataService) PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error) {
	m.lastFileVersion = fileVersion
	return m.previewResult, m.previewErr
}
func (m *mockDataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	return m.applyResult, m.applyErr
}

func TestDataHandler_Export(t *testing.T) {
	mock := &mockDataService{exportEnvelop: &model.BackupEnvelope{App: "ticktask", SchemaVersion: 1, Data: model.BackupData{}}}
	h := NewDataHandler(mock)
	r := setupTestRouter()
	r.GET("/api/data/export", h.Export)

	req, _ := http.NewRequest("GET", "/api/data/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if cd := w.Header().Get("Content-Disposition"); cd == "" || !strings.Contains(cd, "attachment") {
		t.Errorf("missing attachment Content-Disposition: %q", cd)
	}
	var env model.BackupEnvelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if env.App != "ticktask" {
		t.Errorf("body not envelope: %s", w.Body.String())
	}
	// default (no query) → include API key
	if !mock.lastIncludeKey {
		t.Errorf("default export should include API key, got include=%v", mock.lastIncludeKey)
	}
}

func TestDataHandler_Export_IncludeAPIKeyParam(t *testing.T) {
	cases := []struct {
		query      string
		wantInclude bool
	}{
		{"", true},                  // default: include
		{"?include_api_key=true", true},
		{"?include_api_key=false", false},
		{"?include_api_key=0", true}, // only literal "false" disables
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			mock := &mockDataService{exportEnvelop: &model.BackupEnvelope{App: "ticktask", SchemaVersion: 1, Data: model.BackupData{}}}
			h := NewDataHandler(mock)
			r := setupTestRouter()
			r.GET("/api/data/export", h.Export)

			req, _ := http.NewRequest("GET", "/api/data/export"+c.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status %d", w.Code)
			}
			if mock.lastIncludeKey != c.wantInclude {
				t.Errorf("query %q: include=%v, want %v", c.query, mock.lastIncludeKey, c.wantInclude)
			}
		})
	}
}

func TestDataHandler_PreviewImport(t *testing.T) {
	h := NewDataHandler(&mockDataService{previewResult: &model.ImportPreview{Modules: map[string]*model.ModulePreview{"tasks": {New: 1}}}})
	r := setupTestRouter()
	r.POST("/api/data/import/preview", h.PreviewImport)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "b.json")
	env := model.BackupEnvelope{App: "ticktask", SchemaVersion: 1, Data: model.BackupData{Tasks: []model.Task{{ID: "t1"}}}}
	raw, _ := json.Marshal(env)
	fw.Write(raw)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/data/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var prev model.ImportPreview
	json.Unmarshal(w.Body.Bytes(), &prev)
	if prev.Modules["tasks"] == nil || prev.Modules["tasks"].New != 1 {
		t.Errorf("preview not returned: %s", w.Body.String())
	}
}

func TestDataHandler_PreviewImport_BadFile(t *testing.T) {
	h := NewDataHandler(&mockDataService{})
	r := setupTestRouter()
	r.POST("/api/data/import/preview", h.PreviewImport)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "b.json")
	fw.Write([]byte("not json"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/data/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDataHandler_ApplyImport(t *testing.T) {
	h := NewDataHandler(&mockDataService{applyResult: &model.ApplyResult{Applied: map[string]model.ModuleApplyResult{"tasks": {Inserted: 1}}}})
	r := setupTestRouter()
	r.POST("/api/data/import/apply", h.ApplyImport)

	reqBody, _ := json.Marshal(model.ApplyImportRequest{Data: model.BackupData{}, Modules: map[string]model.ModuleApply{}})
	req, _ := http.NewRequest("POST", "/api/data/import/apply", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestDataHandler_ApplyImport_ServiceError(t *testing.T) {
	h := NewDataHandler(&mockDataService{applyErr: errBoom})
	r := setupTestRouter()
	r.POST("/api/data/import/apply", h.ApplyImport)

	reqBody, _ := json.Marshal(model.ApplyImportRequest{Data: model.BackupData{}, Modules: map[string]model.ModuleApply{}})
	req, _ := http.NewRequest("POST", "/api/data/import/apply", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestDataHandler_ApplyImport_InvalidPolicy(t *testing.T) {
	h := NewDataHandler(&mockDataService{applyErr: service.ErrInvalidPolicy})
	r := setupTestRouter()
	r.POST("/api/data/import/apply", h.ApplyImport)

	reqBody, _ := json.Marshal(model.ApplyImportRequest{Data: model.BackupData{}, Modules: map[string]model.ModuleApply{}})
	req, _ := http.NewRequest("POST", "/api/data/import/apply", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid policy, got %d", w.Code)
	}
}
