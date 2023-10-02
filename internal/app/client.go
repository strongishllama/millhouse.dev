package app

import "context"

type Client interface {
	Run(ctx context.Context, cancel context.CancelFunc) error
	Stop(ctx context.Context, cancel context.CancelFunc) error
}
