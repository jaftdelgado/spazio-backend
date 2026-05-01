package uploads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type mockUploadsRepository struct {
	photoID int32
	err     error
	called  bool
	input   SavePhotoInput
}

func (m *mockUploadsRepository) SavePropertyPhoto(_ context.Context, input SavePhotoInput) (int32, error) {
	m.called = true
	m.input = input
	return m.photoID, m.err
}

type mockPhotoStorage struct {
	uploadCalled bool
	deleteCalled bool
	publicCalled bool
	deleteKey    string
	publicURL    string
	uploadErr    error
	deleteErr    error
	publicErr    error
}

func (m *mockPhotoStorage) Upload(_ context.Context, _ string, _ string, _ io.Reader) error {
	m.uploadCalled = true
	return m.uploadErr
}

func (m *mockPhotoStorage) Delete(_ context.Context, storageKey string) error {
	m.deleteCalled = true
	m.deleteKey = storageKey
	return m.deleteErr
}

func (m *mockPhotoStorage) PublicURL(_ context.Context, _ string) (string, error) {
	m.publicCalled = true
	return m.publicURL, m.publicErr
}

func TestUploadPropertyPhotoDeletesOrphanedObjectOnSaveFailure(t *testing.T) {
	repo := &mockUploadsRepository{err: errors.New("db failed")}
	storageClient := &mockPhotoStorage{deleteErr: errors.New("delete failed")}

	svc := NewService(repo, storageClient).(*service)
	svc.encodeToWebP = func(UploadPhotoInput) ([]byte, error) {
		return []byte("webp-bytes"), nil
	}
	result, err := svc.UploadPropertyPhoto(context.Background(), UploadPhotoInput{
		PropertyUUID: "123e4567-e89b-12d3-a456-426614174000",
		MimeType:     "image/png",
		File:         bytes.NewReader([]byte("ignored")),
	})

	if err == nil || !strings.Contains(err.Error(), "save property photo") {
		t.Fatalf("error = %v, want wrapped save property photo error", err)
	}
	if result != (UploadPhotoResult{}) {
		t.Fatalf("result = %#v, want zero value", result)
	}
	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if !storageClient.uploadCalled {
		t.Fatal("expected upload to be called")
	}
	if !storageClient.deleteCalled {
		t.Fatal("expected delete to be called after save failure")
	}
	if storageClient.publicCalled {
		t.Fatal("did not expect public URL lookup after save failure")
	}
	if !strings.Contains(storageClient.deleteKey, "properties/123e4567-e89b-12d3-a456-426614174000/photos/") {
		t.Fatalf("deleteKey = %q, want property photo storage key", storageClient.deleteKey)
	}
}

func TestUploadPropertyPhotoSucceeds(t *testing.T) {
	repo := &mockUploadsRepository{photoID: 9}
	storageClient := &mockPhotoStorage{publicURL: "https://example.com/photo.webp"}

	svc := NewService(repo, storageClient).(*service)
	svc.encodeToWebP = func(UploadPhotoInput) ([]byte, error) {
		return []byte("webp-bytes"), nil
	}
	result, err := svc.UploadPropertyPhoto(context.Background(), UploadPhotoInput{
		PropertyUUID: "123e4567-e89b-12d3-a456-426614174000",
		MimeType:     "image/png",
		File:         bytes.NewReader([]byte("ignored")),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called || !storageClient.uploadCalled || !storageClient.publicCalled {
		t.Fatal("expected repository, upload, and public URL lookups to be called")
	}
	if storageClient.deleteCalled {
		t.Fatal("did not expect delete on success")
	}
	if result.PhotoID != 9 || result.URL != storageClient.publicURL {
		t.Fatalf("result = %#v, want successful upload result", result)
	}
	if result.StorageKey == "" {
		t.Fatal("expected storage key to be populated")
	}
	if repo.input.MimeType != "image/webp" {
		t.Fatalf("repo mime type = %q, want image/webp", repo.input.MimeType)
	}
}

func TestUploadPropertyPhotoDoesNotDeleteOnUploadFailure(t *testing.T) {
	repo := &mockUploadsRepository{}
	storageClient := &mockPhotoStorage{uploadErr: errors.New("upload failed")}

	svc := NewService(repo, storageClient).(*service)
	svc.encodeToWebP = func(UploadPhotoInput) ([]byte, error) {
		return []byte("webp-bytes"), nil
	}

	_, err := svc.UploadPropertyPhoto(context.Background(), UploadPhotoInput{
		PropertyUUID: "123e4567-e89b-12d3-a456-426614174000",
		File:         bytes.NewReader([]byte("ignored")),
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if storageClient.deleteCalled {
		t.Fatal("did not expect delete when upload itself failed")
	}
	if repo.called {
		t.Fatal("did not expect repository call when upload failed")
	}
}

var _ UploadsRepository = (*mockUploadsRepository)(nil)
var _ photoStorage = (*mockPhotoStorage)(nil)
