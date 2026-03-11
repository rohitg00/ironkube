package engine

import "context"

type Pipeline struct {
	phases          []Phase
	state           *State
	completedPhases []string
}

func NewPipeline(phases ...Phase) *Pipeline {
	return &Pipeline{
		phases: phases,
		state:  NewState(),
	}
}

func (p *Pipeline) Execute(ctx context.Context) error {
	for _, phase := range p.phases {
		select {
		case <-ctx.Done():
			p.runCleanups(ctx)
			return ctx.Err()
		default:
		}

		if err := phase.Run(ctx, p.state); err != nil {
			p.runCleanups(ctx)
			return err
		}
		p.completedPhases = append(p.completedPhases, phase.Name())
	}
	return nil
}

func (p *Pipeline) runCleanups(ctx context.Context) {
	for i := len(p.completedPhases) - 1; i >= 0; i-- {
		name := p.completedPhases[i]
		for _, phase := range p.phases {
			if cp, ok := phase.(CleanupPhase); ok && phase.Name() == name {
				_ = cp.Cleanup(ctx, p.state)
			}
		}
	}
}

func (p *Pipeline) CompletedPhases() []string {
	return p.completedPhases
}

func (p *Pipeline) State() *State {
	return p.state
}
