package grpclb

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

type RoundRobinSelector struct {
	baseSelector
	next int
}

func NewRoundRobinSelector() Selector {
	return &RoundRobinSelector{
		next:         0,
		baseSelector: baseSelector{addrMap: make(map[string]*AddrInfo)},
	}
}

func (r *RoundRobinSelector) Get(ctx context.Context) (grpc.Address, error) {
	var (
		addr = grpc.Address{}
		err  error
	)

	if len(r.addrs) == 0 {
		return addr, AddrListEmptyErr
	}

	// reset 0
	if r.next >= len(r.addrs) {
		r.next = 0
	}

	var (
		next  = r.next
		count = len(r.addrs)
	)

	// try run one round
	for index := 0; index < count; index++ {
		a := r.addrs[next]
		next = (next + 1) % len(r.addrs)

		if addrInfo, ok := r.addrMap[a]; ok {
			if !addrInfo.connected {
				continue
			}

			addrInfo.load++
			r.next = next

			return addrInfo.addr, err
		}
	}

	return addr, errors.New("not found available node")
}
