package tests

import (
	"fmt"
	"io/ioutil"
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
	ResponseHeaderTimeout: 1 * time.Second,
}

var client = &http.Client{Transport: transport}

// This assumes the kubectl tool is setup properly for the cluster you'd like to use for testing.

const deployCount = 5
const requestWorkerCount = 10

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

		resp, err := client.Get(url + "/content")
		if err != nil {
			c.Inc(-1)
		} else {
			defer resp.Body.Close()
			c.Inc(resp.StatusCode)
			body, _ := ioutil.ReadAll(resp.Body)
			if string(body) == "I am alive!" {
				t.Log(string(body))
			}
		}
	}
}

// runDeployTest sets up a kubernetes yaml file (should be a deploy + service)
// from the passed in path in a created namespace.  It then runs alot of requests
// against that deployment while re-deploying as soon as a re-deploy completes.
// It returns a map of status code to count.
func runDeployTest(t *testing.T, path string) (results map[int]int) {
	runtime.GOMAXPROCS(requestWorkerCount)

	name := "no-downtime"
	namespace := name + uuid.NewString()

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

	return c.Map
}

// TestNoDowntime checks the effects of having podStop and readiness checks enabled.
func TestNoDowntime(t *testing.T) {
	result := runDeployTest(t, "./deployment_no_downtime.yaml")

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(result)

	assert.Equal(t, len(result), 1, "There should only be one response code type.")
	assert.Greater(t, result[200], 100, "Resp code should be 200 and Greater than 100 requests should have happened")
}

//TestNoDowntimeGraceful checks the effects of having graceful shutdown and readiness checks enabled.
func TestNoDowntimeGraceful(t *testing.T) {
	result := runDeployTest(t, "./deployment_no_downtime_graceful.yaml")

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(result)

	assert.Equal(t, len(result), 1, "There should only be one response code type.")
	assert.Greater(t, result[200], 100, "Resp code should be 200 and Greater than 100 requests should have happened")
}

//TestNoReadiness checks the effects of having podStop but no readiness checks.
func TestNoReadiness(t *testing.T) {
	result := runDeployTest(t, "./deployment_no_readiness.yaml")

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(result)

	assert.Equal(t, 2, len(result), "There should be 2 response code type. (one for conn refused, and 200s)")
	assert.Greater(t, result[200], 0, "There should be some successful requests")
	assert.Greater(t, result[-1], 0, "There should be some request errors")
}

func TestNoPodStop(t *testing.T) {
	result := runDeployTest(t, "./deployment_no_pod_stop.yaml")

	// Check that we have one status code for all the requests, that
	// there were many of them, and that the statusCode is 200.
	fmt.Println("Response code map: ")
	spew.Dump(result)

	assert.Equal(t, 2, len(result), 2, "There should be 2 response code type. (one for conn refused, and 200s)")
	assert.Greater(t, result[200], 0, "There should be some successful requests")
	assert.Greater(t, result[-1], 0, "There should be some request errors")
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
