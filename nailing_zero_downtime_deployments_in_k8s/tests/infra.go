package tests

import (
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// This assumes the kubectl tool is setup properly for the cluster you'd like to use for testing.

const deployCount = 10
const requestWorkerCount = 10
const requestUrl = "localhost:3000"

func deploy(name string, args map[string]string) {
	// Deploy the templated yaml and service
}

type RespRecorder struct {
	sync.RWMutex
	internal map[int]int
}

func NewRespRecorder() *RespRecorder {
	return &RespRecorder{
		internal: make(map[int]int),
	}
}

func (rm *RespRecorder) Increment(key int) {
	rm.RLock()
	if result, ok := rm.internal[key]; !ok {
		rm.internal[key] = 1
	} else {
		rm.internal[key] = result + 1
	}
	rm.RUnlock()
	return
}

// testResponses spins up some workers to send requests to a service, while at the same time calling
// deploy.  It records response codes/counts in a map that is returned.
func testResponses(name string, args string) *RespRecorder {
	// Simultaneosly sending requests request and re-deploying service

	done := false
	respMap := NewRespRecorder()

	wg := &sync.WaitGroup{}
	wg.Add(requestWorkerCount)

	// Spin up workers to send requests
	// Run until the deployment worker markes done var true
	// Then mark wg as done
	for worker := 0; worker < requestWorkerCount; worker++ {
		go func() {
			defer wg.Done()

			for {
				if done { // This is technically a data race, but we don't care
					return
				}

				resp, err := http.Get(requestUrl)
				if err != nil {
					log.Fatalln(err)
				}

				respMap.Increment(resp.StatusCode)
			}
		}()
	}

	// Spin up one worker to run deployments
	go func() {
		for i := 0; i < deployCount; i++ {
			deploy(name, map[string]string{})
		}

		done = true
	}()

	wg.Wait()

	return respMap
}

func TestNoReadiness() {
	name := uuid.NewString()
	args := map[string]string{}
	deploy(name)

	testResponses(name, args)

	// Should have non 200 responses
}

func TestNoGraceful() {

}

func TestNoDowntime() {

}
