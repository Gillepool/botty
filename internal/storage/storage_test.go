package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestStorageGet(t *testing.T) {
	logger := zaptest.NewLogger(t)
	store := NewStorage(logger)

	ok, err := store.Get("test", nil)

	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestStorageSet(t *testing.T) {
	logger := zaptest.NewLogger(t)
	store := NewStorage(logger)

	expected := "result"
	err := store.Set("key", expected)

	assert.NoError(t, err)

	var result string

	ok, err := store.Get("key", &result)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, result)
}
