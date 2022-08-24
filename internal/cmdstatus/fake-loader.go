package cmdstatus

import (
	"context"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/cmd/status"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

type FakeLoader struct {
	ctx       context.Context
	factory   util.Factory
	InvClient *inventory.FakeClient
}

var _ status.Loader = &FakeLoader{}

func NewFakeLoader(ctx context.Context, f util.Factory, objs object.ObjMetadataSet) *FakeLoader {
	return &FakeLoader{
		ctx:       ctx,
		factory:   f,
		InvClient: inventory.NewFakeClient(objs),
	}
}

func (r *FakeLoader) GetInvInfo(cmd *cobra.Command, args []string) (inventory.Info, error) {
	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		args = append(args, cwd)
	}

	path := args[0]
	var err error
	if args[0] != "-" {
		path, err = argutil.ResolveSymlink(r.ctx, path)
		if err != nil {
			return nil, err
		}
	}

	_, inv, err := live.Load(r.factory, path, cmd.InOrStdin())
	if err != nil {
		return nil, err
	}

	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return nil, err
	}

	return invInfo, nil
}
