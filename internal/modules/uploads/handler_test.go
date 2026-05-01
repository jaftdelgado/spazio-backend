package uploads

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"fmt"

	"github.com/gin-gonic/gin"
)

type handlerMockService struct {
	result UploadPhotoResult
	err    error
	called bool
}

func (m *handlerMockService) UploadPropertyPhoto(_ context.Context, _ UploadPhotoInput) (UploadPhotoResult, error) {
	m.called = true
	return m.result, m.err
}

func TestUploadPropertyPhotoNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	handler := NewHandler(&handlerMockService{err: fmt.Errorf("save property photo: %w", ErrPropertyNotFound)})

	body, contentType := newMultipartUploadRequest(t)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/uploads/properties/123e4567-e89b-12d3-a456-426614174000/photos", body)
	ctx.Request.Header.Set("Content-Type", contentType)
	ctx.Params = gin.Params{{Key: "property_uuid", Value: "123e4567-e89b-12d3-a456-426614174000"}}

	handler.uploadPropertyPhoto(ctx)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("property not found")) {
		t.Fatalf("body = %q, want property not found", recorder.Body.String())
	}
}

func TestUploadPropertyPhotoSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	mock := &handlerMockService{result: UploadPhotoResult{PhotoID: 7, StorageKey: "k", URL: "u"}}
	handler := NewHandler(mock)

	body, contentType := newMultipartUploadRequest(t)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/uploads/properties/123e4567-e89b-12d3-a456-426614174000/photos", body)
	ctx.Request.Header.Set("Content-Type", contentType)
	ctx.Params = gin.Params{{Key: "property_uuid", Value: "123e4567-e89b-12d3-a456-426614174000"}}

	handler.uploadPropertyPhoto(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if !mock.called {
		t.Fatal("expected service to be called")
	}
}

func newMultipartUploadRequest(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="photo.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write([]byte("png-data")); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return &body, writer.FormDataContentType()
}
