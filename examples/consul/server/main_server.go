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

	"github.com/rfyiamcool/grpc-scheduler/examples/proto"
	"github.com/rfyiamcool/grpc-scheduler/registry/consul"
)

var nodeID = flag.String("node", "node1", "node ID")
var port = flag.Int("port", 8080, "listening port")

type RpcServer struct {
	addr string
	s    *grpc.Server
}

func NewRpcServer(addr string) *RpcServer {
	s := grpc.NewServer()
	rs := &RpcServer{
		addr: addr,
		s:    s,
	}
	return rs
}

func (s *RpcServer) Run() {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	log.Printf("rpc listening on:%s", s.addr)

	proto.RegisterTestServer(s.s, s)
	s.s.Serve(listener)
}

func (s *RpcServer) Stop() {
	s.s.GracefulStop()
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
			ServiceName: "test",
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
		registry.Register()
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

//go run main.go -node node1 -port 28544
//go run main.go -node node2 -port 18562
//go run main.go -node node3 -port 27772
func main() {
	flag.Parse()
	StartService()
}
