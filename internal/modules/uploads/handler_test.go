package uploads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockUploadsService struct {
	uploadPropertyPhotoFunc  func(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error)
	uploadPropertyPhotosFunc func(ctx context.Context, input UploadPhotosInput) (UploadPhotosBatchResult, error)
	lastInput                UploadPhotoInput
	lastBatchInput           UploadPhotosInput
	lastCtx                  context.Context
}

func (m *mockUploadsService) UploadPropertyPhoto(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
	m.lastCtx = ctx
	m.lastInput = input
	if m.uploadPropertyPhotoFunc != nil {
		return m.uploadPropertyPhotoFunc(ctx, input)
	}
	return UploadPhotoResult{}, nil
}

func (m *mockUploadsService) UploadPropertyPhotos(ctx context.Context, input UploadPhotosInput) (UploadPhotosBatchResult, error) {
	m.lastCtx = ctx
	m.lastBatchInput = input
	if m.uploadPropertyPhotosFunc != nil {
		return m.uploadPropertyPhotosFunc(ctx, input)
	}
	return UploadPhotosBatchResult{}, nil
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func buildMultipartRequest(t *testing.T, propertyUUID string, fileContent []byte, mimeType string, extraFields map[string]string) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// file field (only when fileContent provided)
	if fileContent != nil {
		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Disposition", `form-data; name="file"; filename="photo"`)
		if mimeType != "" {
			hdr.Set("Content-Type", mimeType)
		} else {
			hdr.Set("Content-Type", "application/octet-stream")
		}
		fw, err := w.CreatePart(hdr)
		if err != nil {
			t.Fatalf("create form file part: %v", err)
		}
		if _, err := io.Copy(fw, bytes.NewReader(fileContent)); err != nil {
			t.Fatalf("copy file: %v", err)
		}
	}

	for k, v := range extraFields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}

	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/properties/"+propertyUUID+"/photos", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	// set a fake content type for the file header — Gin reads header from the uploaded file
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func buildMultipartRequestMulti(t *testing.T, propertyUUID string, files []struct {
	content  []byte
	mimeType string
}) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for _, file := range files {
		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Disposition", `form-data; name="file"; filename="photo"`)
		if file.mimeType != "" {
			hdr.Set("Content-Type", file.mimeType)
		} else {
			hdr.Set("Content-Type", "application/octet-stream")
		}
		fw, err := w.CreatePart(hdr)
		if err != nil {
			t.Fatalf("create form file part: %v", err)
		}
		if _, err := io.Copy(fw, bytes.NewReader(file.content)); err != nil {
			t.Fatalf("copy file: %v", err)
		}
	}

	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/properties/"+propertyUUID+"/photos/batch", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func assertHasKey(t *testing.T, rec *httptest.ResponseRecorder, key string) {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := m[key]; !ok {
		t.Fatalf("expected key %q in response, body=%s", key, rec.Body.String())
	}
}

func TestHandler_uploadPropertyPhoto_PropertyUUIDValidation(t *testing.T) {
	cases := []struct {
		name            string
		propertyUUID    string
		fileContent     []byte
		mimeType        string
		wantStatus      int
		wantErrorSubstr string
	}{
		{"missing", "", []byte("x"), "image/jpeg", http.StatusBadRequest, "property_uuid is required"},
		{"invalid uuid", "not-a-uuid", []byte("x"), "image/jpeg", http.StatusBadRequest, "must be a valid UUID"},
		{"no file", uuid.New().String(), nil, "", http.StatusBadRequest, "file is required"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := buildMultipartRequest(t, tt.propertyUUID, tt.fileContent, tt.mimeType, nil)
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = req
			ginCtx.Params = gin.Params{{Key: "property_uuid", Value: tt.propertyUUID}}

			svc := &mockUploadsService{}
			h := NewHandler(svc)
			h.uploadPropertyPhoto(ginCtx)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status got %d want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			assertHasKey(t, rec, "error")
			if !strings.Contains(rec.Body.String(), tt.wantErrorSubstr) {
				t.Fatalf("expected error to contain %q, got %s", tt.wantErrorSubstr, rec.Body.String())
			}
		})
	}
}

func TestHandler_uploadPropertyPhoto_FileValidation(t *testing.T) {
	validUUID := uuid.New().String()

	// prepare contents
	fiveMB := make([]byte, 5*1024*1024)
	fiveMBPlus := make([]byte, 5*1024*1024+1)

	cases := []struct {
		name         string
		content      []byte
		mimeType     string
		wantStatus   int
		wantContains string
	}{
		{"exact 5MB", fiveMB, "image/jpeg", http.StatusCreated, ""},
		{"5MB+1 byte", fiveMBPlus, "image/jpeg", http.StatusBadRequest, "5MB"},
		{"jpeg allowed", []byte("jpeg"), "image/jpeg", http.StatusCreated, ""},
		{"png allowed", []byte("png"), "image/png", http.StatusCreated, ""},
		{"webp allowed", []byte("webp"), "image/webp", http.StatusCreated, ""},
		{"gif not allowed", []byte("gif"), "image/gif", http.StatusBadRequest, "allowed MIME types"},
		{"pdf not allowed", []byte("pdf"), "application/pdf", http.StatusBadRequest, "allowed MIME types"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := buildMultipartRequest(t, validUUID, tt.content, tt.mimeType, nil)
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = req
			ginCtx.Params = gin.Params{{Key: "property_uuid", Value: validUUID}}

			called := false
			svc := &mockUploadsService{
				uploadPropertyPhotoFunc: func(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
					called = true
					return UploadPhotoResult{PhotoID: 1, StorageKey: "properties/" + validUUID + "/photos/1.webp", URL: "https://cdn.example.com/photo.webp"}, nil
				},
			}
			h := NewHandler(svc)
			h.uploadPropertyPhoto(ginCtx)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status got %d want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusCreated {
				assertHasKey(t, rec, "photo_id")
				assertHasKey(t, rec, "storage_key")
				assertHasKey(t, rec, "url")
				if !called {
					t.Fatalf("service was not called")
				}
			} else {
				assertHasKey(t, rec, "error")
				if tt.wantContains != "" && !strings.Contains(rec.Body.String(), tt.wantContains) {
					t.Fatalf("expected error to contain %q, got %s", tt.wantContains, rec.Body.String())
				}
			}
		})
	}
}

func TestHandler_uploadPropertyPhoto_OptionalFieldsParsing(t *testing.T) {
	validUUID := uuid.New().String()
	file := []byte("data")

	cases := []struct {
		name        string
		fields      map[string]string
		wantLabel   *string
		wantAlt     *string
		wantSort    int32
		wantIsCover bool
	}{
		{"none", nil, nil, nil, 0, false},
		{"label", map[string]string{"label": "My Photo"}, strPtr("My Photo"), nil, 0, false},
		{"alt", map[string]string{"alt_text": "A house"}, nil, strPtr("A house"), 0, false},
		{"label whitespace", map[string]string{"label": "   "}, nil, nil, 0, false},
		{"sort_order", map[string]string{"sort_order": "3"}, nil, nil, 3, false},
		{"sort_order invalid", map[string]string{"sort_order": "abc"}, nil, nil, 0, false},
		{"is_cover true", map[string]string{"is_cover": "true"}, nil, nil, 0, true},
		{"is_cover false", map[string]string{"is_cover": "false"}, nil, nil, 0, false},
		{"is_cover invalid", map[string]string{"is_cover": "abc"}, nil, nil, 0, false},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := buildMultipartRequest(t, validUUID, file, "image/jpeg", tt.fields)
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = req
			ginCtx.Params = gin.Params{{Key: "property_uuid", Value: validUUID}}

			var captured UploadPhotoInput
			svc := &mockUploadsService{
				uploadPropertyPhotoFunc: func(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
					captured = input
					return UploadPhotoResult{PhotoID: 1, StorageKey: "k", URL: "u"}, nil
				},
			}
			h := NewHandler(svc)
			h.uploadPropertyPhoto(ginCtx)

			if rec.Code != http.StatusCreated {
				t.Fatalf("status got %d want %d body=%s", rec.Code, http.StatusCreated, rec.Body.String())
			}

			if (captured.Label == nil) != (tt.wantLabel == nil) {
				t.Fatalf("label mismatch: got %v want %v", captured.Label, tt.wantLabel)
			}
			if captured.Label != nil && *captured.Label != *tt.wantLabel {
				t.Fatalf("label content mismatch: got %v want %v", *captured.Label, *tt.wantLabel)
			}
			if (captured.AltText == nil) != (tt.wantAlt == nil) {
				t.Fatalf("alt mismatch: got %v want %v", captured.AltText, tt.wantAlt)
			}
			if captured.AltText != nil && *captured.AltText != *tt.wantAlt {
				t.Fatalf("alt content mismatch: got %v want %v", *captured.AltText, *tt.wantAlt)
			}
			if captured.SortOrder != tt.wantSort {
				t.Fatalf("sort order mismatch: got %v want %v", captured.SortOrder, tt.wantSort)
			}
			if captured.IsCover != tt.wantIsCover {
				t.Fatalf("is_cover mismatch: got %v want %v", captured.IsCover, tt.wantIsCover)
			}
		})
	}
}

func TestHandler_uploadPropertyPhoto_ServiceErrors(t *testing.T) {
	validUUID := uuid.New().String()
	file := []byte("data")

	cases := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantHasErr bool
	}{
		{"not found", ErrPropertyNotFound, http.StatusNotFound, true},
		{"generic", errors.New("boom"), http.StatusInternalServerError, true},
		{"success", nil, http.StatusCreated, false},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := buildMultipartRequest(t, validUUID, file, "image/jpeg", nil)
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = req
			ginCtx.Params = gin.Params{{Key: "property_uuid", Value: validUUID}}

			svc := &mockUploadsService{}
			if tt.serviceErr != nil {
				svc.uploadPropertyPhotoFunc = func(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
					return UploadPhotoResult{}, tt.serviceErr
				}
			} else {
				svc.uploadPropertyPhotoFunc = func(ctx context.Context, input UploadPhotoInput) (UploadPhotoResult, error) {
					return UploadPhotoResult{PhotoID: 1, StorageKey: "k", URL: "u"}, nil
				}
			}

			h := NewHandler(svc)
			h.uploadPropertyPhoto(ginCtx)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status got %d want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantHasErr {
				assertHasKey(t, rec, "error")
			} else {
				assertHasKey(t, rec, "photo_id")
				assertHasKey(t, rec, "storage_key")
				assertHasKey(t, rec, "url")
			}
		})
	}
}

func TestHandler_uploadPropertyPhotos_FileValidation(t *testing.T) {
	validUUID := uuid.New().String()
	fiveMBPlus := make([]byte, 5*1024*1024+1)

	cases := []struct {
		name  string
		files []struct {
			content  []byte
			mimeType string
		}
		wantStatus      int
		wantErrorSubstr string
	}{
		{"0 files", nil, http.StatusBadRequest, "at least one file is required"},
		{"11 files", []struct {
			content  []byte
			mimeType string
		}{
			{[]byte("1"), "image/jpeg"},
			{[]byte("2"), "image/jpeg"},
			{[]byte("3"), "image/jpeg"},
			{[]byte("4"), "image/jpeg"},
			{[]byte("5"), "image/jpeg"},
			{[]byte("6"), "image/jpeg"},
			{[]byte("7"), "image/jpeg"},
			{[]byte("8"), "image/jpeg"},
			{[]byte("9"), "image/jpeg"},
			{[]byte("10"), "image/jpeg"},
			{[]byte("11"), "image/jpeg"},
		}, http.StatusBadRequest, "maximum 10 files allowed"},
		{"invalid mime at index 2", []struct {
			content  []byte
			mimeType string
		}{
			{[]byte("1"), "image/jpeg"},
			{[]byte("2"), "image/png"},
			{[]byte("3"), "image/gif"},
		}, http.StatusBadRequest, "file[2]"},
		{"file too large at index 0", []struct {
			content  []byte
			mimeType string
		}{
			{fiveMBPlus, "image/jpeg"},
			{[]byte("2"), "image/png"},
		}, http.StatusBadRequest, "file[0]"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := buildMultipartRequestMulti(t, validUUID, tt.files)
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = req
			ginCtx.Params = gin.Params{{Key: "property_uuid", Value: validUUID}}

			svc := &mockUploadsService{}
			h := NewHandler(svc)
			h.uploadPropertyPhotos(ginCtx)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status got %d want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			assertHasKey(t, rec, "error")
			if !strings.Contains(rec.Body.String(), tt.wantErrorSubstr) {
				t.Fatalf("expected error to contain %q, got %s", tt.wantErrorSubstr, rec.Body.String())
			}
		})
	}
}

func TestHandler_uploadPropertyPhotos_ResponseStatuses(t *testing.T) {
	validUUID := uuid.New().String()

	cases := []struct {
		name  string
		files []struct {
			content  []byte
			mimeType string
		}
		serviceResult     UploadPhotosBatchResult
		serviceErr        error
		wantStatus        int
		wantUploadedCount int
		wantFailedCount   int
	}{
		{"1 valid file", []struct {
			content  []byte
			mimeType string
		}{
			{[]byte("1"), "image/jpeg"},
		}, UploadPhotosBatchResult{
			Uploaded: []UploadPhotoResult{{PhotoID: 1, StorageKey: "k1", URL: "u1"}},
		}, nil, http.StatusCreated, 1, 0},
		{"10 valid files", []struct {
			content  []byte
			mimeType string
		}{
			{[]byte("1"), "image/jpeg"},
			{[]byte("2"), "image/jpeg"},
			{[]byte("3"), "image/jpeg"},
			{[]byte("4"), "image/jpeg"},
			{[]byte("5"), "image/jpeg"},
			{[]byte("6"), "image/jpeg"},
			{[]byte("7"), "image/jpeg"},
			{[]byte("8"), "image/jpeg"},
			{[]byte("9"), "image/jpeg"},
			{[]byte("10"), "image/jpeg"},
		}, UploadPhotosBatchResult{
			Uploaded: []UploadPhotoResult{
				{PhotoID: 1, StorageKey: "k1", URL: "u1"},
				{PhotoID: 2, StorageKey: "k2", URL: "u2"},
				{PhotoID: 3, StorageKey: "k3", URL: "u3"},
				{PhotoID: 4, StorageKey: "k4", URL: "u4"},
				{PhotoID: 5, StorageKey: "k5", URL: "u5"},
				{PhotoID: 6, StorageKey: "k6", URL: "u6"},
				{PhotoID: 7, StorageKey: "k7", URL: "u7"},
				{PhotoID: 8, StorageKey: "k8", URL: "u8"},
				{PhotoID: 9, StorageKey: "k9", URL: "u9"},
				{PhotoID: 10, StorageKey: "k10", URL: "u10"},
			},
		}, nil, http.StatusCreated, 10, 0},
		{"service fails 1 of 3", []struct {
			content  []byte
			mimeType string
		}{
			{[]byte("1"), "image/jpeg"},
			{[]byte("2"), "image/png"},
			{[]byte("3"), "image/webp"},
		}, UploadPhotosBatchResult{
			Uploaded: []UploadPhotoResult{
				{PhotoID: 1, StorageKey: "k1", URL: "u1"},
				{PhotoID: 3, StorageKey: "k3", URL: "u3"},
			},
			Failed: []UploadPhotoError{{Index: 1, Message: "convert to webp: boom"}},
		}, nil, http.StatusMultiStatus, 2, 1},
		{"service fails all", []struct {
			content  []byte
			mimeType string
		}{
			{[]byte("1"), "image/jpeg"},
			{[]byte("2"), "image/png"},
			{[]byte("3"), "image/webp"},
		}, UploadPhotosBatchResult{
			Failed: []UploadPhotoError{
				{Index: 0, Message: "boom 0"},
				{Index: 1, Message: "boom 1"},
				{Index: 2, Message: "boom 2"},
			},
		}, errors.New("all uploads failed"), http.StatusInternalServerError, 0, 3},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := buildMultipartRequestMulti(t, validUUID, tt.files)
			rec := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(rec)
			ginCtx.Request = req
			ginCtx.Params = gin.Params{{Key: "property_uuid", Value: validUUID}}

			svc := &mockUploadsService{
				uploadPropertyPhotosFunc: func(ctx context.Context, input UploadPhotosInput) (UploadPhotosBatchResult, error) {
					return tt.serviceResult, tt.serviceErr
				},
			}
			h := NewHandler(svc)
			h.uploadPropertyPhotos(ginCtx)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status got %d want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			assertHasKey(t, rec, "uploaded")
			var body UploadPhotosBatchResult
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if len(body.Uploaded) != tt.wantUploadedCount {
				t.Fatalf("uploaded count got %d want %d", len(body.Uploaded), tt.wantUploadedCount)
			}
			if len(body.Failed) != tt.wantFailedCount {
				t.Fatalf("failed count got %d want %d", len(body.Failed), tt.wantFailedCount)
			}
		})
	}
}

func strPtr(s string) *string { t := s; return &t }
