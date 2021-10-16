package tests

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/assert"
)

var transport http.RoundTripper = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	ResponseHeaderTimeout: time.Second,
}

var client = &http.Client{Transport: transport}

// This assumes the kubectl tool is setup properly for the cluster you'd like to use for testing.

const deployCount = 20
const requestWorkerCount = 10

// deploy creates namespace for deployment, and runs deploy.
func deploy(t *testing.T, name, namespace, path string) (url string) {

	// Setup the kubectl config and context.
	options := k8s.NewKubectlOptions("", "", namespace)

	// Run `kubectl apply` to deploy. Fail the test if there are any errors.
	k8s.KubectlApply(t, options, path)

	// Verify the service is available and get the URL for it.
	k8s.WaitUntilServiceAvailable(t, options, name, 10, 10*time.Second)
	service := k8s.GetService(t, options, name)
	url = fmt.Sprintf("http://%s", k8s.GetServiceEndpoint(t, options, service, 80))

	return url
}

func startRequester(c *StatusCodeCounter, url string, doneMutex *Done, wg *sync.WaitGroup) {
	for {
		if doneMutex.IsDone() {
			fmt.Println("Done!")
			return
		}

		resp, err := client.Get(url)
		if err != nil {
			log.Fatalln(err)
		}

		c.Inc(resp.StatusCode)
	}
}

func TestNoDowntime(t *testing.T) {
	t.Parallel()

	name := "no-downtime"
	namespace := name + uuid.NewString()
	path := "./deployment_no_downtime.yaml"

	// Onetime setup
	options := k8s.NewKubectlOptions("", "", namespace)
	k8s.CreateNamespace(t, options, namespace)
	defer k8s.DeleteNamespace(t, options, namespace)

	// Deploy and wait for it to be active
	url := deploy(t, name, namespace, path)

	doneMutex := NewDone()
	c := &StatusCodeCounter{}

	workerWg := &sync.WaitGroup{}
	workerWg.Add(requestWorkerCount)

	for i := 0; i < requestWorkerCount; i++ {
		go startRequester(c, url, doneMutex, workerWg)
	}

	go func() {
		for i := 0; i < deployCount; i++ {
			// Redeploy while we're running requests, to see if any requests fail
			_ = deploy(t, name, namespace, path)
		}

		doneMutex.SetDone()
	}()

	// Workers wait on deploys to finish.  Wait here for workers to finish
	workerWg.Wait()

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(c.Map)

	assert.Equal(t, len(c.Map), 1, "There should only be one response code type.")
	for k, v := range c.Map {
		assert.Equal(t, k, 200, "Status Code response should be 200")
		assert.Greater(t, v, 100, "Greater than 100 requests should have happened")
	}
}

func TestNoReadiness(t *testing.T) {
}

func TestNoGraceful(t *testing.T) {

}

// StatusCodeCounter counts the number of status codes.  Concurrent safe.
type StatusCodeCounter struct {
	sync.RWMutex
	Map map[int]int
}

// Inc increases the value in the RWMap for a key.
//   This is more pleasant than r.Set(key, r.Get(key)++)
func (r *StatusCodeCounter) Inc(key int) {
	r.Lock()
	defer r.Unlock()
	r.Map[key]++
}

// Done is a struct for handling a boolean flag
type Done struct {
	sync.RWMutex
	*sync.WaitGroup
	val bool
}

func NewDone() *Done {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	return &Done{
		sync.RWMutex{},
		wg,
		false,
	}
}

func (d *Done) SetDone() {
	d.Lock()
	defer d.Unlock()

	d.Done()

	return
}

func (d *Done) IsDone() bool {
	d.RLock()
	defer d.RUnlock()

	return d.val
}

func (d *Done) Wait() {
	d.Wait()
}
