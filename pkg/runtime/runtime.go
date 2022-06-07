package runtime

import (
	"context"

	"github.com/open-feature/flagd/pkg/provider"
	"github.com/open-feature/flagd/pkg/service"
)

func Start(server service.IService, provider provider.IProvider, ctx context.Context) {
	go server.Serve(provider)
}
