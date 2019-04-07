package consul

import (
	// "encoding/json"
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/naming"
)

var (
	defaultWatchWaitTime = 15 * time.Second

	ErrFetchNodesTimeout = errors.New("fetch nodes raise timeout in consul")
)

// ConsulWatcher is the implementation of grpc.naming.Watcher
type ConsulWatcher struct {
	ctx context.Context
	sync.RWMutex
	running     bool
	serviceName string
	client      *api.Client
	addrs       []*naming.Update
	waitTimeout time.Duration

	addrsMap  map[string]struct{}
	lastIndex uint64
}

func newConsulWatcher(serviceName string, client *api.Client, ctx context.Context) naming.Watcher {
	w := &ConsulWatcher{
		serviceName: serviceName,
		running:     true,
		client:      client,
		ctx:         ctx,
		waitTimeout: defaultWatchWaitTime,
	}

	return w
}

func (w *ConsulWatcher) Close() {
	w.running = false
}

func (w *ConsulWatcher) handleNext() ([]*naming.Update, error) {
	var (
		updates    []*naming.Update
		updateChan = make(chan []*naming.Update)
		errChan    = make(chan error)
	)

	// var ctx context.Context
	// if w.waitTimeout == 0 {
	// 	ctx = w.ctx
	// } else {
	// 	ctx, _ = context.WithTimeout(w.ctx, w.waitTimeout)
	// }

	go w.fetchNodes(w.ctx, updateChan, errChan)

	select {
	case updates = <-updateChan:
		return updates, nil

	case err := <-errChan:
		return nil, err

	case <-w.ctx.Done():
		return updates, nil
	}
}

func (w *ConsulWatcher) Next() ([]*naming.Update, error) {
	return w.handleNext()
}

func (w *ConsulWatcher) fetchNodes(ctx context.Context, updateChan chan []*naming.Update, errChan chan error) ([]*naming.Update, error) {
	for w.running {
		var (
			updates []*naming.Update
			addrs   = map[string]struct{}{}
		)

		queryOptions := &api.QueryOptions{
			WaitIndex: w.lastIndex,   // wait event
			WaitTime:  w.waitTimeout, // while: fetch with timeout, don't break
		}
		queryOptions.WithContext(ctx)

		services, metainfo, err := w.client.Health().Service(
			w.serviceName,
			"",           // tag
			true,         // passing
			queryOptions, // opt
		)
		if err != nil {
			errChan <- err
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
			updateChan <- updates
			return updates, nil
		}
	}

	var updates = []*naming.Update{}
	updateChan <- updates
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
