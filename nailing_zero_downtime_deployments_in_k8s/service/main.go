package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	arg "github.com/alexflint/go-arg"
)

type Settings struct {
	Port     int `arg:"required"`
	Graceful bool
}

var args = Settings{}

func healthz(s Settings) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "I am alive!\n")
	}
}

func content(s Settings) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Lorem ipsum dolor...\n")
	}
}

func startServer(s Settings) http.Server {
	http.HandleFunc("/healthz", healthz(s))
	http.HandleFunc("/contetn", content(s))

	srv := http.Server{
		Addr: ":" + strconv.Itoa(s.Port),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen:%+s\n", err)
		}
	}()

	return srv
}

// gracefulShutdown waits for SIGTERM or SIGQUIT
// then waits 10 seconds (since traffic should have stopped)
// it then stops keepalive connections, waits another 10 seconds
// and returns (should be end of program)
func gracefulShutdown(srv http.Server) {
	sc := make(chan os.Signal, 1)

	signal.Notify(sc,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	<-sc

	fmt.Println("Terminating after 10s")
	// After recieving signal, wait 10 seconds then exit
	// Better to have some mechanism in place for checking with other parts to make sure
	// they're ready to shutdown
	time.Sleep(10 * time.Second)
	srv.SetKeepAlivesEnabled(false)
	time.Sleep(10 * time.Second)
}

func main() {
	arg.MustParse(&args)

	// Delay to simulate some non-zero startup time
	time.Sleep(10 * time.Second)

	log.Printf("Starting on port: %v Graceful: %v", args.Port, args.Graceful)
	srv := startServer(args)

	if args.Graceful {
		fmt.Println("Graceful shutdown enabled")

		gracefulShutdown(srv)
	} else {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		wg.Wait()
	}
}
