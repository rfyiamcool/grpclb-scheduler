package consul

import (
	"errors"

	capi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/naming"
)

// ConsulResolver is the implementaion of grpc.naming.Resolver
type ConsulResolver struct {
	client      *capi.Client
	serviceName string
}

// NewResolver return ConsulResolver with service name
func NewResolver(serviceName string, consulConfig *capi.Config) (*ConsulResolver, error) {
	client, err := capi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return &ConsulResolver{
		serviceName: serviceName,
		client:      client,
	}, nil
}

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
	watcher := newConsulWatcher(c.serviceName, c.client)
	return watcher, nil
}
