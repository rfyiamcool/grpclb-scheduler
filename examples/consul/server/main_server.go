package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	capi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/rfyiamcool/grpclb-scheduler/examples/proto"
	"github.com/rfyiamcool/grpclb-scheduler/registry/consul"
)

var (
	nodeID      = flag.String("node", "node1", "node ID")
	port        = flag.Int("port", 8080, "listening port")
	ServiceName = "test"
)

type RpcServer struct {
	addr   string
	server *grpc.Server
}

func NewRpcServer(addr string) *RpcServer {
	s := grpc.NewServer()
	rs := &RpcServer{
		addr:   addr,
		server: s,
	}

	proto.RegisterTestServer(s, rs)

	healthCheck := health.NewServer()
	healthCheck.SetServingStatus(
		ServiceName,
		grpc_health_v1.HealthCheckResponse_SERVING,
	)
	grpc_health_v1.RegisterHealthServer(s, healthCheck)

	return rs
}

func (s *RpcServer) Run() {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}

	log.Printf("rpc listening on:%s", s.addr)
	s.server.Serve(listener)
}

func (s *RpcServer) Stop() {
	s.server.GracefulStop()
}

func (s *RpcServer) Say(ctx context.Context, req *proto.SayReq) (*proto.SayResp, error) {
	text := "Hello " + req.Content + ", I am " + *nodeID
	log.Println(text)

	return &proto.SayResp{Content: text}, nil
}

func StartService() {
	config := &capi.Config{
		Address: "http://127.0.0.1:8500",
	}

	registry, err := consul.NewRegistry(
		&consul.Congfig{
			ConsulCfg:   config,
			ServiceName: ServiceName,
			NData: consul.NodeData{
				ID:      *nodeID,
				Address: "127.0.0.1",
				Port:    *port,
				// Metadata: map[string]string{"weight": "1"},
			},
			TTL: 5,
		})
	if err != nil {
		log.Panic(err)
		return
	}

	server := NewRpcServer(fmt.Sprintf("0.0.0.0:%d", *port))
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		registry.RegisterGRPCHealth()
		// registry.Register()
		wg.Done()
	}()

	//stop the server after one minute
	//go func() {
	//	time.Sleep(time.Minute)
	//	server.Stop()
	//	registry.Deregister()
	//}()

	wg.Wait()
}

func main() {
	flag.Parse()
	StartService()
}
