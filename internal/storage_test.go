package storage

import "testing"

func TestStorageGet(t *testing.T) {
	store := NewStorage()

	ok, err := store.Get("test", nil)

	if err != nil {
		t.Errorf("err was incorrect")
	}
	if ok != false {
		t.Errorf("Ok was incorrect, got: %t, want: %t.", ok, false)
	}
}
