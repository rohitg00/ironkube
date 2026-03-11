package engine

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_SetAndGet(t *testing.T) {
	s := NewState()
	s.Set("key1", "value1")
	s.Set("key2", 42)

	v1, ok1 := s.Get("key1")
	require.True(t, ok1)
	assert.Equal(t, "value1", v1)

	v2, ok2 := s.Get("key2")
	require.True(t, ok2)
	assert.Equal(t, 42, v2)
}

func TestState_GetMissing(t *testing.T) {
	s := NewState()

	v, ok := s.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, v)
}

func TestState_Overwrite(t *testing.T) {
	s := NewState()
	s.Set("key", "original")

	v1, ok1 := s.Get("key")
	require.True(t, ok1)
	assert.Equal(t, "original", v1)

	s.Set("key", "updated")

	v2, ok2 := s.Get("key")
	require.True(t, ok2)
	assert.Equal(t, "updated", v2)
}

func TestState_ConcurrentAccess(t *testing.T) {
	s := NewState()
	var wg sync.WaitGroup
	iterations := 1000

	for i := range iterations {
		wg.Add(2)

		go func(val int) {
			defer wg.Done()
			s.Set("counter", val)
		}(i)

		go func() {
			defer wg.Done()
			s.Get("counter")
		}()
	}

	wg.Wait()

	_, ok := s.Get("counter")
	assert.True(t, ok)
}
