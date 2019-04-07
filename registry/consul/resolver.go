package consul

import (
	"context"
	"errors"
	"time"

	capi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/naming"
)

// ConsulResolver is the implementaion of grpc.naming.Resolver
type ConsulResolver struct {
	ctx    context.Context
	cancel context.CancelFunc

	waitTime    time.Duration
	client      *capi.Client
	serviceName string
}

// NewResolver return ConsulResolver with service name
func NewResolver(serviceName string, consulConfig *capi.Config) (*ConsulResolver, error) {
	client, err := capi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConsulResolver{
		serviceName: serviceName,
		client:      client,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// NewResolverWithClient
func NewResolverWithClient(serviceName string, client *capi.Client) *ConsulResolver {
	return &ConsulResolver{
		serviceName: serviceName,
		client:      client,
	}
}

// Resolve to resolve the service from consul
func (c *ConsulResolver) Resolve(target string) (naming.Watcher, error) {
	if c.serviceName == "" {
		return nil, errors.New("no service name provided")
	}

	// return ConsulWatcher
	watcher := newConsulWatcher(c.serviceName, c.client, c.ctx)
	return watcher, nil
}
