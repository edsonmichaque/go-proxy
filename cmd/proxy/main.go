package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
)

func main() {
	log.Fatal(proxy([]int{1234, 1235, 1236}, []string{"127.0.0.1:12340", "127.0.0.1:12350"}))
}

func proxy(ports []int, targets []string) error {
	var tcpListeners []net.Listener
	for _, port := range ports {
		tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return err
		}

		tcpListeners = append(tcpListeners, tcpListener)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for _, l := range tcpListeners {
		go func(lis net.Listener) {
			r := &RoundRobin{
				targets: targets,
			}

			src, err := lis.Accept()
			if err != nil {
				panic(err)
			}

			go handleConn(src, r.CurrentTarget())
		}(l)
	}

	<-sig

	return nil
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
