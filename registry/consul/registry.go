package consul

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	capi "github.com/hashicorp/consul/api"

	"github.com/rfyiamcool/grpclb-scheduler/log"
)

var (
	hostName, _ = os.Hostname()
)

type ConsulRegistry struct {
	ctx     context.Context
	cancel  context.CancelFunc
	client  *capi.Client
	cfg     *Congfig
	checkID string
}

type Congfig struct {
	ConsulCfg   *capi.Config
	ServiceName string
	NData       NodeData
	TTL         int // ttl seconds
}

type NodeData struct {
	ID       string
	Address  string
	Port     int
	Metadata map[string]string
}

func NewRegistry(cfg *Congfig) (*ConsulRegistry, error) {
	// validate
	if cfg.NData.ID == "" {
		cfg.NData.ID = makeServerId()
	}
	if cfg.ServiceName == "" {
		return nil, errors.New("invalid service name")
	}

	c, err := capi.NewClient(cfg.ConsulCfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ConsulRegistry{
		ctx:     ctx,
		cancel:  cancel,
		client:  c,
		cfg:     cfg,
		checkID: "service:" + cfg.NData.ID,
	}, nil
}

func (c *ConsulRegistry) RegisterGRPCHealth() error {
	checker := &capi.AgentServiceCheck{
		Interval:                       fmt.Sprintf("%ds", c.cfg.TTL/2),
		GRPC:                           fmt.Sprintf("%v:%v/%v", c.cfg.NData.Address, c.cfg.NData.Port, c.cfg.ServiceName),
		DeregisterCriticalServiceAfter: "1m",
	}

	_, err := c.handleRegister(checker)
	return err
}

func (c *ConsulRegistry) Register() error {
	checker := &capi.AgentServiceCheck{
		TTL:                            fmt.Sprintf("%ds", c.cfg.TTL),
		Status:                         capi.HealthPassing,
		DeregisterCriticalServiceAfter: "1m",
	}

	regisger, err := c.handleRegister(checker)
	if err != nil {
		return err
	}

	keepAliveTicker := time.NewTicker(time.Duration(c.cfg.TTL) * time.Second / 3)
	registerTicker := time.NewTicker(time.Minute)

	for {
		select {
		case <-c.ctx.Done():
			keepAliveTicker.Stop()
			registerTicker.Stop()
			c.client.Agent().ServiceDeregister(c.cfg.NData.ID)
			return nil

		case <-keepAliveTicker.C:
			err := c.client.Agent().PassTTL(c.checkID, "")
			if err != nil {
				log.DefaultLogger("consul registry check %v.\n", err)
			}

		case <-registerTicker.C:
			err = regisger()
			if err != nil {
				log.DefaultLogger("consul register service error: %v.\n", err)
			}
		}
	}
}

func (c *ConsulRegistry) handleRegister(checker *capi.AgentServiceCheck) (func() error, error) {
	// register service
	metadata, err := json.Marshal(c.cfg.NData.Metadata)
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0)
	tags = append(tags, string(metadata))
	serviceID := fmt.Sprintf("%s-%s-%v-%v:%v",
		c.cfg.NData.ID,
		c.cfg.ServiceName,
		hostName,
		c.cfg.NData.Address, c.cfg.NData.Port,
	)

	registerHandler := func() error {
		regisDto := &capi.AgentServiceRegistration{
			ID:      serviceID,
			Name:    c.cfg.ServiceName,
			Address: c.cfg.NData.Address,
			Port:    c.cfg.NData.Port,
			Tags:    tags,
			Check:   checker,
		}
		err := c.client.Agent().ServiceRegister(regisDto)
		if err != nil {
			return fmt.Errorf("register service to consul error: %s\n", err.Error())
		}

		return nil
	}

	err = registerHandler()
	if err != nil {
		return nil, err
	}

	return registerHandler, nil
}

func (c *ConsulRegistry) Deregister() error {
	c.cancel()
	return nil
}

func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

func makeServerId() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%v", uint64(rand.Int63()))
}
