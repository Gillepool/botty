package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

type Storage struct {
	mu      sync.RWMutex
	memory  Memory
	encoder MemoryEncoder
}

// The Memory interface allows the bot to persist data as key-value pairs.
// The default implementation of the Memory is to store all keys and values in
// a map (i.e. in-memory). Other implementations typically offer actual long term
// persistence into a file or to redis.
type Memory interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, bool, error)
	Delete(key string) (bool, error)
	Keys() ([]string, error)
	Close() error
}

// A MemoryEncoder is used to encode and decode any values that are stored in
// the Memory. The default implementation that is used by the Storage uses a
// JSON encoding.
type MemoryEncoder interface {
	Encode(value interface{}) ([]byte, error)
	Decode(data []byte, target interface{}) error
}

type inMemory struct {
	data map[string][]byte
}

type jsonEncoder struct{}

func NewStorage() *Storage {
	return &Storage{
		memory:  newInMemory(),
		encoder: new(jsonEncoder),
	}
}

// Close closes the Memory that is managed by this Storage.
func (s *Storage) Close() error {
	s.mu.Lock()
	err := s.memory.Close()
	s.mu.Unlock()
	return err
}

func (s *Storage) Set(key string, value interface{}) error {
	data, err := s.encoder.Encode(value)
	if err != nil {
		return fmt.Errorf("Failed to encode value %w ", err)
	}
	s.mu.Lock()
	err = s.memory.Set(key, data)
	s.mu.Unlock()
	return err
}

func (s *Storage) Get(key string, value interface{}) (bool, error) {
	s.mu.RLock()
	data, ok, err := s.memory.Get(key)
	s.mu.RUnlock()
	if err != nil {
		return false, fmt.Errorf("Failed to fetch value %w ", err)
	}
	if value == nil {
		return ok, nil
	}

	err = s.encoder.Decode(data, value)
	if err != nil {
		return false, fmt.Errorf("Failed to Decode value %w ", err)
	}
	return true, nil
}

func (s *Storage) Delete(key string) (bool, error) {
	s.mu.Lock()
	ok, err := s.memory.Delete(key)
	s.mu.Unlock()
	return ok, err
}

func (s *Storage) Keys() ([]string, error) {
	s.mu.RLock()
	keys, err := s.memory.Keys()
	s.mu.RUnlock()

	sort.Strings(keys)
	return keys, err
}

func (s *Storage) SetMemory(m Memory) {
	s.mu.RLock()
	s.memory = m
	s.mu.RUnlock()
}

func (s *Storage) SetMemoryEncoder(memoryEncoder MemoryEncoder) {
	s.mu.RLock()
	s.encoder = memoryEncoder
	s.mu.RUnlock()
}

func (m *inMemory) Close() error {
	m.data = map[string][]byte{}
	return nil
}

func newInMemory() *inMemory {
	return &inMemory{data: map[string][]byte{}}
}

func (m *inMemory) Delete(key string) (bool, error) {
	_, ok := m.data[key]
	delete(m.data, key)
	return ok, nil
}

func (m *inMemory) Set(key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *inMemory) Get(key string) ([]byte, bool, error) {
	value, ok := m.data[key]
	return value, ok, nil
}

func (m *inMemory) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}

	return keys, nil
}

func (jsonEncoder) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonEncoder) Decode(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}
