package uploads

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type mockUploadsRepository struct {
	savePropertyPhotoFunc func(ctx context.Context, input SavePhotoInput) (int32, error)
	lastInput             SavePhotoInput
}

func (m *mockUploadsRepository) SavePropertyPhoto(ctx context.Context, input SavePhotoInput) (int32, error) {
	m.lastInput = input
	if m.savePropertyPhotoFunc != nil {
		return m.savePropertyPhotoFunc(ctx, input)
	}
	return 0, nil
}

type mockPhotoStorage struct {
	uploadFunc    func(ctx context.Context, storageKey string, contentType string, body io.Reader) error
	deleteFunc    func(ctx context.Context, storageKey string) error
	publicURLFunc func(ctx context.Context, storageKey string) (string, error)

	calledUpload   bool
	calledDelete   bool
	calledPublic   bool
	lastUploadBody []byte
	lastKey        string
}

func (m *mockPhotoStorage) Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error {
	m.calledUpload = true
	m.lastKey = storageKey
	if body != nil {
		b, _ := io.ReadAll(body)
		m.lastUploadBody = b
	}
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, storageKey, contentType, bytes.NewReader(m.lastUploadBody))
	}
	return nil
}

func (m *mockPhotoStorage) Delete(ctx context.Context, storageKey string) error {
	m.calledDelete = true
	m.lastKey = storageKey
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, storageKey)
	}
	return nil
}

func (m *mockPhotoStorage) PublicURL(ctx context.Context, storageKey string) (string, error) {
	m.calledPublic = true
	m.lastKey = storageKey
	if m.publicURLFunc != nil {
		return m.publicURLFunc(ctx, storageKey)
	}
	return "", nil
}

func TestService_UploadPropertyPhoto_HappyPath(t *testing.T) {
	propUUID := uuid.New().String()

	repo := &mockUploadsRepository{}
	storage := &mockPhotoStorage{}

	encodeCalled := false
	encodeFn := func(input UploadPhotoInput) ([]byte, error) {
		encodeCalled = true
		return []byte("webpdata"), nil
	}

	s := &service{repository: repo, r2Client: storage, encodeToWebP: encodeFn}

	_, err := s.UploadPropertyPhoto(context.Background(), UploadPhotoInput{PropertyUUID: propUUID, File: bytes.NewReader([]byte("dummy"))})

	if !encodeCalled {
		t.Fatalf("expected encodeToWebP to be called")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !encodeCalled {
		t.Fatalf("encodeToWebP was not called")
	}
	if !storage.calledUpload {
		t.Fatalf("r2 upload was not called")
	}
	if repo.lastInput.StorageKey == "" {
		t.Fatalf("repository SavePropertyPhoto not called with StorageKey")
	}
	// repo returned 0 by default; no PhotoID assertions needed here
	// assert storage key structure
	if !strings.HasPrefix(repo.lastInput.StorageKey, "properties/"+propUUID+"/photos/") {
		t.Fatalf("storage key prefix mismatch: %s", repo.lastInput.StorageKey)
	}
	if !strings.HasSuffix(repo.lastInput.StorageKey, ".webp") {
		t.Fatalf("storage key suffix mismatch: %s", repo.lastInput.StorageKey)
	}
}

func TestService_UploadPropertyPhoto_FailurePaths(t *testing.T) {
	repoErr := errors.New("db fail")

	cases := []struct {
		name                string
		encodeErr           error
		uploadErr           error
		saveErr             error
		publicURLErr        error
		deleteErr           error
		expectErrContains   string
		expectUploadCalled  bool
		expectSaveCalled    bool
		expectDeleteCalled  bool
		expectPublicCalled  bool
		expectIsNotFoundErr bool
	}{
		{
			name:               "encode error",
			encodeErr:          errors.New("bad image"),
			expectErrContains:  "convert to webp",
			expectUploadCalled: false,
		},
		{
			name:               "r2 upload error",
			uploadErr:          errors.New("r2 fail"),
			expectErrContains:  "upload to r2",
			expectUploadCalled: true,
			expectSaveCalled:   false,
		},
		{
			name:               "repository save error triggers delete",
			saveErr:            repoErr,
			expectErrContains:  "save property photo",
			expectUploadCalled: true,
			expectSaveCalled:   true,
			expectDeleteCalled: true,
		},
		{
			name:                "repository ErrPropertyNotFound triggers delete and wraps",
			saveErr:             ErrPropertyNotFound,
			expectErrContains:   "property not found",
			expectUploadCalled:  true,
			expectSaveCalled:    true,
			expectDeleteCalled:  true,
			expectIsNotFoundErr: true,
		},
		{
			name:               "repository error and delete fails",
			saveErr:            repoErr,
			deleteErr:          errors.New("delete fail"),
			expectErrContains:  "save property photo",
			expectUploadCalled: true,
			expectSaveCalled:   true,
			expectDeleteCalled: true,
		},
		{
			name:               "public url error",
			publicURLErr:       errors.New("no url"),
			expectErrContains:  "get public url",
			expectUploadCalled: true,
			expectSaveCalled:   true,
			expectPublicCalled: true,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			propUUID := uuid.New().String()
			repo := &mockUploadsRepository{}
			storage := &mockPhotoStorage{}

			// configure mocks
			if tt.uploadErr != nil {
				storage.uploadFunc = func(ctx context.Context, storageKey string, contentType string, body io.Reader) error {
					return tt.uploadErr
				}
			}
			if tt.deleteErr != nil {
				storage.deleteFunc = func(ctx context.Context, storageKey string) error {
					return tt.deleteErr
				}
			}
			if tt.publicURLErr != nil {
				storage.publicURLFunc = func(ctx context.Context, storageKey string) (string, error) {
					return "", tt.publicURLErr
				}
			} else {
				storage.publicURLFunc = func(ctx context.Context, storageKey string) (string, error) {
					return "https://cdn.example.com/photo.webp", nil
				}
			}

			if tt.saveErr != nil {
				repo.savePropertyPhotoFunc = func(ctx context.Context, input SavePhotoInput) (int32, error) {
					return 0, tt.saveErr
				}
			} else {
				repo.savePropertyPhotoFunc = func(ctx context.Context, input SavePhotoInput) (int32, error) {
					return 1, nil
				}
			}

			encodeCalled := false
			encodeFn := func(input UploadPhotoInput) ([]byte, error) {
				encodeCalled = true
				if tt.encodeErr != nil {
					return nil, tt.encodeErr
				}
				return []byte("webp"), nil
			}
			_ = encodeCalled

			s := &service{repository: repo, r2Client: storage, encodeToWebP: encodeFn}

			_, err := s.UploadPropertyPhoto(context.Background(), UploadPhotoInput{PropertyUUID: propUUID, File: bytes.NewReader([]byte("dummy"))})

			if tt.expectIsNotFoundErr {
				if !errors.Is(err, ErrPropertyNotFound) {
					t.Fatalf("expected ErrPropertyNotFound, got %v", err)
				}
				// still expect delete called
				if !storage.calledDelete {
					t.Fatalf("expected delete called on not found")
				}
				return
			}

			if tt.expectErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectErrContains)
				}
				if !strings.Contains(err.Error(), tt.expectErrContains) {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if storage.calledUpload != tt.expectUploadCalled {
				t.Fatalf("upload called=%v want=%v", storage.calledUpload, tt.expectUploadCalled)
			}
			if repo.savePropertyPhotoFunc != nil {
				// if save func provided we consider save called when SavePropertyPhoto executed
				if (repo.lastInput.StorageKey != "") != tt.expectSaveCalled {
					t.Fatalf("save called=%v want=%v", repo.lastInput.StorageKey != "", tt.expectSaveCalled)
				}
			}
			if storage.calledDelete != tt.expectDeleteCalled {
				t.Fatalf("delete called=%v want=%v", storage.calledDelete, tt.expectDeleteCalled)
			}
			if storage.calledPublic != tt.expectPublicCalled {
				t.Fatalf("public called=%v want=%v", storage.calledPublic, tt.expectPublicCalled)
			}
		})
	}
}
