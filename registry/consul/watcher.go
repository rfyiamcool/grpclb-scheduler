package consul

import (
	// "encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/naming"
)

// ConsulWatcher is the implementation of grpc.naming.Watcher
type ConsulWatcher struct {
	sync.RWMutex
	running     bool
	serviceName string
	client      *api.Client
	updateChan  chan []*naming.Update
	addrs       []*naming.Update

	addrsMap  map[string]struct{}
	lastIndex uint64
}

func newConsulWatcher(serviceName string, client *api.Client) naming.Watcher {
	w := &ConsulWatcher{
		serviceName: serviceName,
		updateChan:  make(chan []*naming.Update),
		running:     true,
		client:      client,
	}

	return w
}

func (w *ConsulWatcher) Close() {
	w.running = false
	close(w.updateChan)
}

func (w *ConsulWatcher) Next() ([]*naming.Update, error) {
	fmt.Println(51)
	// select {
	// case updates := <-w.updates:
	// 	return updates, nil
	// }

	nodes, err := w.FetchNodes()
	fmt.Println(52)
	fmt.Println(nodes, err)
	return nodes, err
	// return []*naming.Update{}, nil
}

func (w *ConsulWatcher) FetchNodes() ([]*naming.Update, error) {
	for w.running {
		var (
			updates []*naming.Update
			addrs   = map[string]struct{}{}
		)

		services, metainfo, err := w.client.Health().Service(
			w.serviceName,
			"",   // tag
			true, // passing
			&api.QueryOptions{
				WaitIndex: w.lastIndex, // wait event
			},
		)
		if err != nil {
			grpclog.Println("error retrieving instances from Consul: %v", err)
			return nil, err
		}

		w.lastIndex = metainfo.LastIndex
		for _, service := range services {
			addrs[net.JoinHostPort(service.Service.Address, strconv.Itoa(service.Service.Port))] = struct{}{}
		}

		// set operation DELETE flag; delete not passing nodes
		for addr := range w.addrsMap {
			if _, ok := addrs[addr]; !ok {
				updates = append(updates, &naming.Update{Op: naming.Delete, Addr: addr})
			}
		}

		// set operation ADD flag; add new nodes
		for addr := range addrs {
			if _, ok := w.addrsMap[addr]; !ok {
				updates = append(updates, &naming.Update{Op: naming.Add, Addr: addr})
			}
		}

		if len(updates) != 0 {
			w.addrsMap = addrs
			return updates, nil
		}
	}

	var updates = []*naming.Update{}
	return updates, nil
}

// func (w *ConsulWatcher) handle(idx uint64, data interface{}) {
// 	entries, ok := data.([]*api.ServiceEntry)
// 	if !ok {
// 		return
// 	}

// 	addrs := []*naming.Update{}

// 	for _, e := range entries {
// 		for _, check := range e.Checks {
// 			if check.ServiceID == e.Service.ID {
// 				if check.Status == api.HealthPassing {
// 					addr := fmt.Sprintf("%s:%d", e.Service.Address, e.Service.Port)
// 					metadata := map[string]string{}
// 					if len(e.Service.Tags) > 0 {
// 						err := json.Unmarshal([]byte(e.Service.Tags[0]), &metadata)
// 						if err != nil {
// 							grpclog.Println("Parse node data error:", err)
// 						}
// 					}
// 					addrs = append(addrs, &naming.Update{Addr: addr, Metadata: &metadata})
// 				}
// 				break
// 			}
// 		}
// 	}

// 	updates := []*naming.Update{}

// 	//add new address
// 	for _, newAddr := range addrs {
// 		found := false
// 		for _, oldAddr := range w.addrs {
// 			if newAddr.Addr == oldAddr.Addr {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			newAddr.Op = naming.Add
// 			updates = append(updates, newAddr)
// 		}
// 	}

// 	//delete old address
// 	for _, oldAddr := range w.addrs {
// 		found := false
// 		for _, addr := range addrs {
// 			if addr.Addr == oldAddr.Addr {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			oldAddr.Op = naming.Delete
// 			updates = append(updates, oldAddr)
// 		}
// 	}
// 	w.addrs = addrs

// 	w.updatesChan <- updates
// }
