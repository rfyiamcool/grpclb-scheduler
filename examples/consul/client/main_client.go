package main

import (
	"context"
	"log"

	capi "github.com/hashicorp/consul/api"
	"google.golang.org/grpc"

	"github.com/rfyiamcool/grpclb-scheduler"
	"github.com/rfyiamcool/grpclb-scheduler/examples/proto"
	"github.com/rfyiamcool/grpclb-scheduler/registry/consul"
)

func main() {
	config := &capi.Config{
		Address: "http://127.0.0.1:8500",
	}

	resolver, err := consul.NewResolver("test", config)
	if err != nil {
		panic(err.Error())
	}

	b := grpclb.NewBalancer(resolver, grpclb.NewRoundRobinSelector())
	conn, err := grpc.Dial("", grpc.WithInsecure(), grpc.WithBalancer(b))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer conn.Close()

	client := proto.NewTestClient(conn)

	var count = 10
	for index := 0; index < count; index++ {
		resp, err := client.Say(context.Background(), &proto.SayReq{Content: "consul"})
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf(resp.Content)
	}
}
