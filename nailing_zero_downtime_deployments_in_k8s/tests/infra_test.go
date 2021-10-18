package tests

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
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
	ResponseHeaderTimeout: 3 * time.Second,
}

var client = &http.Client{Transport: transport}

// This assumes the kubectl tool is setup properly for the cluster you'd like to use for testing.

const deployCount = 30
const requestWorkerCount = 50

// deploy creates namespace for deployment, and runs deploy.
func deploy(t *testing.T, name, namespace, path string) (url string) {

	// Setup the kubectl config and context.
	options := k8s.NewKubectlOptions("", "", namespace)

	// Run `kubectl apply` to deploy. Fail the test if there are any errors.
	k8s.KubectlApply(t, options, path)

	// Verify the service is available and get the URL for it.
	k8s.WaitUntilServiceAvailable(t, options, name, 10, 20*time.Second)
	service := k8s.GetService(t, options, name)
	url = fmt.Sprintf("http://%s/healthz", k8s.GetServiceEndpoint(t, options, service, 80))

	return url
}

// Rollout restart triggers a restart of the service.  This is the same as a deploy with the exception
// that we're not changing the image.  Bit hacky since terratest doesn't seem to have a way to do rollout restart.
func restart(t *testing.T, name, namespace string) {
	depName := "deployment/" + name
	out, err := exec.Command("kubectl", "rollout", "restart", "--namespace", namespace, depName).Output()
	if err != nil && strings.Contains(string(out), "deployment.apps/"+name+" restarted") {
		t.Log("Error rolling out restart: " + string(out))

		t.Fail()
	}

	// This command by default waits for rollout to complete before returning
	out, err = exec.Command("kubectl", "rollout", "status", "--namespace", namespace, depName).Output()
	if err != nil && strings.Contains(string(out), "deployment.apps/"+name+" restarted") {
		t.Log("Status")
		t.Log("Error rolling out restart: " + string(out))

		t.Fail()
	}
}

func startRequester(t *testing.T, c *StatusCodeCounter, url string, doneMutex *Done, wg *sync.WaitGroup) {
	for {
		if doneMutex.IsDone() {
			wg.Done()
			return
		}

		resp, err := client.Get(url)
		if err != nil {
			t.Log(err, resp.StatusCode)
			c.Inc(-1)
		} else {
			resp.Body.Close()
			c.Inc(resp.StatusCode)
		}
	}
}

func TestNoDowntime(t *testing.T) {
	runtime.GOMAXPROCS(requestWorkerCount)

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
	c := NewStatusCodeCounter()

	workerWg := &sync.WaitGroup{}
	workerWg.Add(requestWorkerCount)

	for i := 0; i < requestWorkerCount; i++ {
		go startRequester(t, c, url, doneMutex, workerWg)
	}

	for i := 0; i < deployCount; i++ {
		// Redeploy while we're running requests, to see if any requests fail
		restart(t, name, namespace)
	}

	doneMutex.SetDone()

	// Workers wait on deploys to finish.  Wait here for workers to finish
	workerWg.Wait()

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(c.Map)

	assert.Equal(t, len(c.Map), 1, "There should only be one response code type.")
	assert.Greater(t, c.Map[200], 100, "Greater than 100 requests should have happened")
}

func TestNoReadiness(t *testing.T) {
	runtime.GOMAXPROCS(requestWorkerCount)

	name := "no-downtime"
	namespace := name + uuid.NewString()
	path := "./deployment_no_readiness.yaml"

	// Onetime setup
	options := k8s.NewKubectlOptions("", "", namespace)
	k8s.CreateNamespace(t, options, namespace)
	defer k8s.DeleteNamespace(t, options, namespace)

	// Deploy and wait for it to be active
	url := deploy(t, name, namespace, path)

	doneMutex := NewDone()
	c := NewStatusCodeCounter()

	workerWg := &sync.WaitGroup{}
	workerWg.Add(requestWorkerCount)

	for i := 0; i < requestWorkerCount; i++ {
		go startRequester(t, c, url, doneMutex, workerWg)
	}

	for i := 0; i < deployCount; i++ {
		// Redeploy while we're running requests, to see if any requests fail
		restart(t, name, namespace)
	}

	doneMutex.SetDone()

	// Workers wait on deploys to finish.  Wait here for workers to finish
	workerWg.Wait()

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(c.Map)

	assert.Equal(t, 2, len(c.Map), "There should be 2 response code type. (one for conn refused, and 200s)")
	assert.Greater(t, c.Map[200], 0, "There should be some successful requests")
}

func TestNoPodStop(t *testing.T) {
	runtime.GOMAXPROCS(requestWorkerCount)

	name := "no-downtime"
	namespace := name + uuid.NewString()
	path := "./deployment_no_pod_stop.yaml"

	// Onetime setup
	options := k8s.NewKubectlOptions("", "", namespace)
	k8s.CreateNamespace(t, options, namespace)
	defer k8s.DeleteNamespace(t, options, namespace)

	// Deploy and wait for it to be active
	url := deploy(t, name, namespace, path)

	doneMutex := NewDone()
	c := NewStatusCodeCounter()

	workerWg := &sync.WaitGroup{}
	workerWg.Add(requestWorkerCount)

	for i := 0; i < requestWorkerCount; i++ {
		go startRequester(t, c, url, doneMutex, workerWg)
	}

	for i := 0; i < deployCount; i++ {
		// Redeploy while we're running requests, to see if any requests fail
		restart(t, name, namespace)
	}

	doneMutex.SetDone()

	// Workers wait on deploys to finish.  Wait here for workers to finish
	workerWg.Wait()

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(c.Map)

	assert.Equal(t, 2, len(c.Map), 2, "There should be 2 response code type. (one for conn refused, and 200s)")
	assert.Greater(t, c.Map[200], 0, "There should be some successful requests")
}

// StatusCodeCounter counts the number of status codes.  Concurrent safe.
type StatusCodeCounter struct {
	sync.RWMutex
	Map map[int]int
}

func NewStatusCodeCounter() *StatusCodeCounter {
	return &StatusCodeCounter{
		RWMutex: sync.RWMutex{},
		Map:     map[int]int{},
	}
}

// Inc increases the value in the RWMap for a key.
func (r *StatusCodeCounter) Inc(key int) {
	r.Lock()
	defer r.Unlock()

	if v, ok := r.Map[key]; !ok {
		r.Map[key] = 1
	} else {
		r.Map[key] = v + 1
	}
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

	d.val = true
	d.Done()

	return
}

func (d *Done) IsDone() bool {
	d.RLock()
	defer d.RUnlock()

	return d.val
}
