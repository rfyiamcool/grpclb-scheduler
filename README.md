# grpclb-schduler
This is a gRPC load balancer for go, supports Random, RoundRobin and consistent hashing strategies.

![](https://raw.githubusercontent.com/liyue201/grpc-lb/master/struct.png "")

grpc-schduler's code base on [github.com/liyue201/grpc-lb](github.com/liyue201/grpc-lb)

## Modify

* format code
* custom logger
* timeout control
* add nginx's smooth weighted round-robin balancing
* add grpc health check
* fix consul serverEntry interface conv bug
* deincr health interval
* more ...
