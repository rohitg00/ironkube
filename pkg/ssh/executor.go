package ssh

import (
	"context"
	"fmt"
	"sync"
)

type Result struct {
	Host   string
	Output string
	Err    error
}

type Executor struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewExecutor() *Executor {
	return &Executor{
		clients: make(map[string]*Client),
	}
}

func (e *Executor) Connect(cfg Config) error {
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.clients[cfg.Host] = client
	return nil
}

func (e *Executor) RunOnAll(ctx context.Context, cmd string) []Result {
	e.mu.RLock()
	hosts := make([]string, 0, len(e.clients))
	for h := range e.clients {
		hosts = append(hosts, h)
	}
	e.mu.RUnlock()

	return e.RunOnHosts(ctx, hosts, cmd)
}

func (e *Executor) RunOnHosts(ctx context.Context, hosts []string, cmd string) []Result {
	results := make([]Result, len(hosts))
	var wg sync.WaitGroup

	for i, host := range hosts {
		wg.Add(1)
		go func(idx int, h string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				results[idx] = Result{
					Host: h,
					Err:  ctx.Err(),
				}
				return
			default:
			}

			output, err := e.RunOnHost(h, cmd)
			results[idx] = Result{
				Host:   h,
				Output: output,
				Err:    err,
			}
		}(i, host)
	}

	wg.Wait()
	return results
}

func (e *Executor) RunOnHost(host, cmd string) (string, error) {
	e.mu.RLock()
	client, ok := e.clients[host]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("no connection for host %s", host)
	}

	return client.Run(cmd)
}

func (e *Executor) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, client := range e.clients {
		client.Close()
	}
	e.clients = make(map[string]*Client)
}
