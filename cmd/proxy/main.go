package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

func main() {
	log.Fatal(proxy(1234, []string{"127.0.0.1:12340", "127.0.0.1:12350"}))
}

func proxy(port int, targets []string) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	r := &RoundRobin{
		targets: targets,
	}

	for {
		src, err := l.Accept()
		if err != nil {
			continue
		}

		go handleConn(src, r.CurrentTarget())
	}
}

func handleConn(src net.Conn, addr string) {
	defer func() {
		log.Println("closing src")
		src.Close()
	}()

	dst, err := net.Dial("tcp", addr)
	if err != nil {
		panic(dst)
	}

	defer func() {
		log.Println("closing dst")
		dst.Close()
	}()

	for {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(dst, src)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			io.Copy(src, dst)
		}()

		wg.Wait()
	}
}

type balancingMode interface {
	CurrentTarget() string
}

type RoundRobin struct {
	targets []string
	cur     int
	mu      sync.Mutex
}

func (r *RoundRobin) CurrentTarget() string {
	target := r.targets[r.cur]

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cur = (r.cur % len(r.targets)) + 1

	return target
}
