package eventbuss

import (
	"context"

	"github.com/rafaeljesus/rabbus"
)

type Option func(*EventBuss) error

func Context(ctx context.Context) Option {
	return func(r *EventBuss) error {
		r.ctx = ctx
		return nil
	}
}

func Service(service string) Option {
	return func(r *EventBuss) error {
		r.service = service
		return nil
	}
}

func SetConfig(config map[Event]rabbus.ListenConfig) Option {
	return func(r *EventBuss) error {
		r.config = config
		return nil
	}
}

func Verbose() Option {
	return func(r *EventBuss) error {
		r.verbose = true
		return nil
	}
}
