package phases

import (
	"context"
	"fmt"

	"github.com/rohitg00/ironkube/pkg/config"
	"github.com/rohitg00/ironkube/pkg/distro"
	"github.com/rohitg00/ironkube/pkg/engine"
	"github.com/rohitg00/ironkube/pkg/security"
)

type ValidatePhase struct {
	Config *config.ClusterConfig
}

func (p *ValidatePhase) Name() string { return "validate" }

func (p *ValidatePhase) Run(ctx context.Context, state *engine.State) error {
	if err := config.Validate(p.Config); err != nil {
		return err
	}

	plugin, err := distro.Get(p.Config.Spec.Distro)
	if err != nil {
		return err
	}

	if err := plugin.ValidateVersion(p.Config.Spec.Version); err != nil {
		return fmt.Errorf("version validation: %w", err)
	}

	_, err = security.Get(p.Config.Spec.Security.Profile)
	if err != nil {
		return err
	}

	state.Set("distro_plugin", plugin)
	return nil
}
