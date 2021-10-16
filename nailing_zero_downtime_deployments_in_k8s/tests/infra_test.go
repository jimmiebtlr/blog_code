package tests

import (
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"

)

// This assumes the kubectl tool is setup properly for the cluster you'd like to use for testing.

const deployCount = 20
const requestWorkerCount = 10


func deploy(t *testing.T, name, path string) (url string) {
	// Setup the kubectl config and context.
	options := k8s.NewKubectlOptions("", "", "default")

	// Run `kubectl apply` to deploy. Fail the test if there are any errors.
	k8s.KubectlApply(t, options, path)

	// Verify the service is available and get the URL for it.
	k8s.WaitUntilServiceAvailable(t, options, name, 10, 1*time.Second)
	service := k8s.GetService(t, options, name)
	url := fmt.Sprintf("http://%s", k8s.GetServiceEndpoint(t, options, service, 5000))

	return url
}

func startReqester(c *StatusCodeCounter, url string, doneMutex *Done) {
	for {
		if doneMutex.IsDone() {
			return
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}

		c.Increment(resp.StatusCode)
	}
}


func TestNoDowntime(t *testing.T) {
	t.Parallel()

	path := "./deployment_no_downtime.yml"
	url := deploy(t, path)
	
	doneMutex := NewDone()
	c := &StatusCodeCounter{}

	for i := 0; i < requestWorkerCount; i++ {
		go startRequester(c, url, doneMutex)
	}

	go func() {
		for i := 0; i < deployCount; i++ {
			_ := deploy(t, path)
		}	

		doneMutex.SetDone()
	} () 


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



// StatusCodeCounter counts the number of status codes.  Concurrent safe.
type StatusCodeCounter struct {
    sync.RWMutex
    Map map[int]int
}

// Inc increases the value in the RWMap for a key.
//   This is more pleasant than r.Set(key, r.Get(key)++)
func (r StatusCodeCounter) Inc(key int) {
    r.Lock()
    defer r.Unlock()
    r.m[key]++
}


// Done is a struct for handling a boolean flag
type Done struct {
	*sync.RWMutex{}
	val bool
}

func NewDone() *Done {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	return &Done{
		&sync.Mutex{},
		wg,
	}
}

func (d *Done) SetDone() {
	d.WLock() 
	defer d.WUnlock()

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