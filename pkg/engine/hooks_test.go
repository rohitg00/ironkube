package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPreHook struct {
	name string
	fn   func(ctx context.Context, phaseName string, state *State) error
}

func (h *testPreHook) Name() string { return h.name }
func (h *testPreHook) PrePhase(ctx context.Context, phaseName string, state *State) error {
	if h.fn != nil {
		return h.fn(ctx, phaseName, state)
	}
	return nil
}

type testPostHook struct {
	name string
	fn   func(ctx context.Context, phaseName string, state *State, phaseErr error) error
}

func (h *testPostHook) Name() string { return h.name }
func (h *testPostHook) PostPhase(ctx context.Context, phaseName string, state *State, phaseErr error) error {
	if h.fn != nil {
		return h.fn(ctx, phaseName, state, phaseErr)
	}
	return nil
}

type testErrorHook struct {
	name string
	fn   func(ctx context.Context, phaseName string, state *State, err error) error
}

func (h *testErrorHook) Name() string { return h.name }
func (h *testErrorHook) OnError(ctx context.Context, phaseName string, state *State, err error) error {
	if h.fn != nil {
		return h.fn(ctx, phaseName, state, err)
	}
	return nil
}

type testAllHook struct {
	name    string
	preFn   func(ctx context.Context, phaseName string, state *State) error
	postFn  func(ctx context.Context, phaseName string, state *State, phaseErr error) error
	errorFn func(ctx context.Context, phaseName string, state *State, err error) error
}

func (h *testAllHook) Name() string { return h.name }
func (h *testAllHook) PrePhase(ctx context.Context, phaseName string, state *State) error {
	if h.preFn != nil {
		return h.preFn(ctx, phaseName, state)
	}
	return nil
}
func (h *testAllHook) PostPhase(ctx context.Context, phaseName string, state *State, phaseErr error) error {
	if h.postFn != nil {
		return h.postFn(ctx, phaseName, state, phaseErr)
	}
	return nil
}
func (h *testAllHook) OnError(ctx context.Context, phaseName string, state *State, err error) error {
	if h.errorFn != nil {
		return h.errorFn(ctx, phaseName, state, err)
	}
	return nil
}

func TestHook_PrePhaseRunsBeforeEachPhase(t *testing.T) {
	var order []string

	hook := &testPreHook{name: "pre", fn: func(_ context.Context, phaseName string, _ *State) error {
		order = append(order, "pre:"+phaseName)
		return nil
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:a")
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:b")
		return nil
	}}

	p := NewPipeline(a, b).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"pre:a", "run:a", "pre:b", "run:b"}, order)
}

func TestHook_PostPhaseRunsAfterEachPhase(t *testing.T) {
	var order []string

	hook := &testPostHook{name: "post", fn: func(_ context.Context, phaseName string, _ *State, _ error) error {
		order = append(order, "post:"+phaseName)
		return nil
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:a")
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:b")
		return nil
	}}

	p := NewPipeline(a, b).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"run:a", "post:a", "run:b", "post:b"}, order)
}

func TestHook_PostPhaseReceivesNilOnSuccess(t *testing.T) {
	var receivedErr error
	called := false

	hook := &testPostHook{name: "post", fn: func(_ context.Context, _ string, _ *State, phaseErr error) error {
		called = true
		receivedErr = phaseErr
		return nil
	}}

	a := &mockPhase{name: "a"}

	p := NewPipeline(a).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.True(t, called)
	assert.Nil(t, receivedErr)
}

func TestHook_PostPhaseReceivesPhaseError(t *testing.T) {
	errFail := errors.New("phase failed")
	var receivedErr error

	hook := &testPostHook{name: "post", fn: func(_ context.Context, _ string, _ *State, phaseErr error) error {
		receivedErr = phaseErr
		return nil
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		return errFail
	}}

	p := NewPipeline(a).WithHooks(hook)
	_ = p.Execute(context.Background())

	assert.ErrorIs(t, receivedErr, errFail)
}

func TestHook_PrePhaseErrorAbortsPipeline(t *testing.T) {
	errAbort := errors.New("pre-hook abort")
	phaseRan := false

	hook := &testPreHook{name: "blocker", fn: func(_ context.Context, _ string, _ *State) error {
		return errAbort
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		phaseRan = true
		return nil
	}}

	p := NewPipeline(a).WithHooks(hook)
	err := p.Execute(context.Background())

	require.ErrorIs(t, err, errAbort)
	assert.False(t, phaseRan)
	assert.Empty(t, p.CompletedPhases())
}

func TestHook_ErrorHookRunsOnPhaseFailure(t *testing.T) {
	errFail := errors.New("boom")
	var errorHookCalled bool
	var capturedPhaseName string
	var capturedErr error

	hook := &testErrorHook{name: "err-hook", fn: func(_ context.Context, phaseName string, _ *State, err error) error {
		errorHookCalled = true
		capturedPhaseName = phaseName
		capturedErr = err
		return nil
	}}

	a := &mockPhase{name: "failing-phase", runFn: func(_ context.Context, _ *State) error {
		return errFail
	}}

	p := NewPipeline(a).WithHooks(hook)
	_ = p.Execute(context.Background())

	assert.True(t, errorHookCalled)
	assert.Equal(t, "failing-phase", capturedPhaseName)
	assert.ErrorIs(t, capturedErr, errFail)
}

func TestHook_ErrorHookNotCalledOnSuccess(t *testing.T) {
	called := false

	hook := &testErrorHook{name: "err-hook", fn: func(_ context.Context, _ string, _ *State, _ error) error {
		called = true
		return nil
	}}

	a := &mockPhase{name: "a"}
	p := NewPipeline(a).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.False(t, called)
}

func TestHook_MultipleHooksExecuteInOrder(t *testing.T) {
	var order []string

	hook1 := &testPreHook{name: "first", fn: func(_ context.Context, _ string, _ *State) error {
		order = append(order, "hook1")
		return nil
	}}
	hook2 := &testPreHook{name: "second", fn: func(_ context.Context, _ string, _ *State) error {
		order = append(order, "hook2")
		return nil
	}}
	hook3 := &testPreHook{name: "third", fn: func(_ context.Context, _ string, _ *State) error {
		order = append(order, "hook3")
		return nil
	}}

	a := &mockPhase{name: "a"}
	p := NewPipeline(a).WithHooks(hook1, hook2, hook3)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"hook1", "hook2", "hook3"}, order)
}

func TestHook_WithHooksReturnsPipelineForChaining(t *testing.T) {
	hook1 := &testPreHook{name: "h1"}
	hook2 := &testPreHook{name: "h2"}

	a := &mockPhase{name: "a"}
	p := NewPipeline(a)
	result := p.WithHooks(hook1).WithHooks(hook2)

	assert.Same(t, p, result)
}

func TestHook_HooksHaveAccessToState(t *testing.T) {
	hook := &testPreHook{name: "state-writer", fn: func(_ context.Context, _ string, state *State) error {
		state.Set("hook_key", "hook_value")
		return nil
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, state *State) error {
		v, ok := state.Get("hook_key")
		if !ok {
			return errors.New("expected hook_key in state")
		}
		if v.(string) != "hook_value" {
			return errors.New("unexpected value")
		}
		return nil
	}}

	p := NewPipeline(a).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
}

func TestHook_PipelineWorksWithoutHooks(t *testing.T) {
	var order []string

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "a")
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "b")
		return nil
	}}

	p := NewPipeline(a, b)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, order)
}

func TestHook_ContextCancellationStillRunsCleanups(t *testing.T) {
	cleanupRan := false
	ctx, cancel := context.WithCancel(context.Background())

	hook := &testPreHook{name: "pre"}

	a := &mockCleanupPhase{
		mockPhase: mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
			cancel()
			return nil
		}},
		cleanupFn: func(_ context.Context, _ *State) error {
			cleanupRan = true
			return nil
		},
	}
	b := &mockPhase{name: "b"}

	p := NewPipeline(a, b).WithHooks(hook)
	err := p.Execute(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, cleanupRan)
}

func TestHook_FullLifecycleOrdering(t *testing.T) {
	var order []string

	hook := &testAllHook{
		name: "lifecycle",
		preFn: func(_ context.Context, phaseName string, _ *State) error {
			order = append(order, "pre:"+phaseName)
			return nil
		},
		postFn: func(_ context.Context, phaseName string, _ *State, _ error) error {
			order = append(order, "post:"+phaseName)
			return nil
		},
	}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:a")
		return nil
	}}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:b")
		return nil
	}}

	p := NewPipeline(a, b).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{
		"pre:a", "run:a", "post:a",
		"pre:b", "run:b", "post:b",
	}, order)
}

func TestHook_ErrorLifecycleOrdering(t *testing.T) {
	errFail := errors.New("fail")
	var order []string

	hook := &testAllHook{
		name: "lifecycle",
		preFn: func(_ context.Context, phaseName string, _ *State) error {
			order = append(order, "pre:"+phaseName)
			return nil
		},
		postFn: func(_ context.Context, phaseName string, _ *State, _ error) error {
			order = append(order, "post:"+phaseName)
			return nil
		},
		errorFn: func(_ context.Context, phaseName string, _ *State, _ error) error {
			order = append(order, "error:"+phaseName)
			return nil
		},
	}

	a := &mockPhase{name: "a"}
	b := &mockPhase{name: "b", runFn: func(_ context.Context, _ *State) error {
		order = append(order, "run:b")
		return errFail
	}}

	p := NewPipeline(a, b).WithHooks(hook)
	err := p.Execute(context.Background())

	require.ErrorIs(t, err, errFail)
	assert.Equal(t, []string{
		"pre:a", "post:a",
		"pre:b", "run:b", "post:b", "error:b",
	}, order)
}

func TestHook_PreHookErrorTriggersCleanup(t *testing.T) {
	errAbort := errors.New("abort")
	cleanupRan := false

	hook := &testPreHook{name: "blocker", fn: func(_ context.Context, phaseName string, _ *State) error {
		if phaseName == "b" {
			return errAbort
		}
		return nil
	}}

	a := &mockCleanupPhase{
		mockPhase: mockPhase{name: "a"},
		cleanupFn: func(_ context.Context, _ *State) error {
			cleanupRan = true
			return nil
		},
	}
	b := &mockPhase{name: "b"}

	p := NewPipeline(a, b).WithHooks(hook)
	err := p.Execute(context.Background())

	require.ErrorIs(t, err, errAbort)
	assert.True(t, cleanupRan)
	assert.Equal(t, []string{"a"}, p.CompletedPhases())
}

func TestHook_PostHookCanReadStateSetByPhase(t *testing.T) {
	var captured any

	hook := &testPostHook{name: "reader", fn: func(_ context.Context, _ string, state *State, _ error) error {
		captured, _ = state.Get("phase_data")
		return nil
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, state *State) error {
		state.Set("phase_data", 42)
		return nil
	}}

	p := NewPipeline(a).WithHooks(hook)
	err := p.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 42, captured)
}

func TestHook_SecondPreHookFailsFirstStillRan(t *testing.T) {
	errAbort := errors.New("second hook fails")
	var order []string

	hook1 := &testPreHook{name: "first", fn: func(_ context.Context, _ string, _ *State) error {
		order = append(order, "hook1")
		return nil
	}}
	hook2 := &testPreHook{name: "second", fn: func(_ context.Context, _ string, _ *State) error {
		order = append(order, "hook2")
		return errAbort
	}}

	a := &mockPhase{name: "a"}
	p := NewPipeline(a).WithHooks(hook1, hook2)
	err := p.Execute(context.Background())

	require.ErrorIs(t, err, errAbort)
	assert.Equal(t, []string{"hook1", "hook2"}, order)
}

func TestHook_MultipleErrorHooksAllCalled(t *testing.T) {
	errFail := errors.New("fail")
	var order []string

	hook1 := &testErrorHook{name: "err1", fn: func(_ context.Context, _ string, _ *State, _ error) error {
		order = append(order, "err1")
		return nil
	}}
	hook2 := &testErrorHook{name: "err2", fn: func(_ context.Context, _ string, _ *State, _ error) error {
		order = append(order, "err2")
		return nil
	}}

	a := &mockPhase{name: "a", runFn: func(_ context.Context, _ *State) error {
		return errFail
	}}

	p := NewPipeline(a).WithHooks(hook1, hook2)
	_ = p.Execute(context.Background())

	assert.Equal(t, []string{"err1", "err2"}, order)
}
