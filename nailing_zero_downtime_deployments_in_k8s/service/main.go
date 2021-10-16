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

func startServer(s Settings) {
	http.HandleFunc("/healthz", healthz(s))
	http.HandleFunc("/contetn", content(s))

	go func() {
		if err := http.ListenAndServe(":"+strconv.Itoa(s.Port), nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen:%+s\n", err)
		}
	}()
}

func main() {
	arg.MustParse(&args)

	log.Printf("Starting on port: %v Graceful: %v", args.Port, args.Graceful)
	startServer(args)

	if args.Graceful {
		sc := make(chan os.Signal, 1)

		signal.Notify(sc,
			os.Interrupt,
			syscall.SIGTERM,
			syscall.SIGQUIT)

		<-sc

		// After recieving signal, wait 10 seconds then exit
		time.Sleep(10 * time.Second)
	} else {
		// Since we don't handle signal, this exits immediatly? Or does it hit max timeout to exit?
		wg := &sync.WaitGroup{}
		wg.Add(1)
		wg.Wait()
	}
}
