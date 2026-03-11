package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPhase struct {
	name   string
	runFn  func(ctx context.Context, state *State) error
}

func (m *mockPhase) Name() string {
	return m.name
}

func (m *mockPhase) Run(ctx context.Context, state *State) error {
	if m.runFn != nil {
		return m.runFn(ctx, state)
	}
	return nil
}

type mockCleanupPhase struct {
	mockPhase
	cleanupFn func(ctx context.Context, state *State) error
}

func (m *mockCleanupPhase) Cleanup(ctx context.Context, state *State) error {
	if m.cleanupFn != nil {
		return m.cleanupFn(ctx, state)
	}
	return nil
}

func TestPipeline_RunsAllPhasesInOrder(t *testing.T) {
	var order []string

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "a")
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "b")
		return nil
	}}
	c := &mockPhase{name: "c", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "c")
		return nil
	}}

	p := NewPipeline(a, b, c)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, order)
	assert.Equal(t, []string{"a", "b", "c"}, p.CompletedPhases())
}

func TestPipeline_StopsOnError(t *testing.T) {
	var order []string
	errFail := errors.New("phase a failed")

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "a")
		return errFail
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "b")
		return nil
	}}

	p := NewPipeline(a, b)
	err := p.Execute(context.Background())

	require.ErrorIs(t, err, errFail)
	assert.Equal(t, []string{"a"}, order)
	assert.Empty(t, p.CompletedPhases())
}

func TestPipeline_CleanupCalledOnFailureInReverseOrder(t *testing.T) {
	var cleanupOrder []string

	a := &mockCleanupPhase{
		mockPhase: mockPhase{name: "a"},
		cleanupFn: func(_ context.Context, _ *State) error {
			cleanupOrder = append(cleanupOrder, "a")
			return nil
		},
	}
	b := &mockCleanupPhase{
		mockPhase: mockPhase{name: "b"},
		cleanupFn: func(_ context.Context, _ *State) error {
			cleanupOrder = append(cleanupOrder, "b")
			return nil
		},
	}
	c := &mockPhase{name: "c", runFn: func(_ context.Context, _ *State) error {
		return errors.New("c failed")
	}}

	p := NewPipeline(a, b, c)
	err := p.Execute(context.Background())

	require.Error(t, err)
	assert.Equal(t, []string{"a", "b"}, p.CompletedPhases())
	assert.Equal(t, []string{"b", "a"}, cleanupOrder)
}

func TestPipeline_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		cancel()
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		return nil
	}}

	p := NewPipeline(a, b)
	err := p.Execute(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"a"}, p.CompletedPhases())
}

func TestPipeline_StateSharedAcrossPhases(t *testing.T) {
	a := &mockPhase{name: "a", runFn: func(_ context.Context, state *State) error {
		state.Set("from_a", "hello")
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, state *State) error {
		v, ok := state.Get("from_a")
		if !ok {
			return errors.New("expected key from_a to exist")
		}
		state.Set("from_b", v.(string)+" world")
		return nil
	}}

	p := NewPipeline(a, b)
	err := p.Execute(context.Background())

	require.NoError(t, err)

	v, ok := p.State().Get("from_b")
	require.True(t, ok)
	assert.Equal(t, "hello world", v)
}

func TestPipeline_CompletedPhasesTracking(t *testing.T) {
	a := &mockPhase{name: "phase-1"}
	b := &mockPhase{name: "phase-2", runFn: func(_ context.Context, _ *State) error {
		return errors.New("fail")
	}}
	c := &mockPhase{name: "phase-3"}

	p := NewPipeline(a, b, c)
	err := p.Execute(context.Background())

	require.Error(t, err)
	assert.Equal(t, []string{"phase-1"}, p.CompletedPhases())
}
