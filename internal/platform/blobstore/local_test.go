package blobstore

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestLocalStoreEnforcesTypeSizeAndPath(t *testing.T) {
	store, err := NewLocal(t.TempDir(), 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(context.Background(), "../escape", "image/png", bytes.NewBufferString("ok")); err == nil {
		t.Fatal("path traversal should be rejected")
	}
	if _, err := store.Save(context.Background(), "bad-type", "application/x-msdownload", bytes.NewBufferString("ok")); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("unsupported type error=%v", err)
	}
	if _, err := store.Save(context.Background(), "too-large", "image/png", bytes.NewBufferString("12345")); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversize error=%v", err)
	}
	saved, err := store.Save(context.Background(), "valid", "image/png", bytes.NewBufferString("1234"))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Size != 4 || saved.SHA256 == "" {
		t.Fatalf("saved=%+v", saved)
	}
}
