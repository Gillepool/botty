package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageGet(t *testing.T) {
	store := NewStorage()

	ok, err := store.Get("test", nil)

	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestStorageSet(t *testing.T) {
	store := NewStorage()

	expected := "result"
	err := store.Set("key", expected)

	assert.NoError(t, err)

	var result string

	ok, err := store.Get("key", &result)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, result)
}
