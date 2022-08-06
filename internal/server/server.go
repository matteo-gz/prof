package server

import (
	"context"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewHTTPServer)

type InterFace interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
