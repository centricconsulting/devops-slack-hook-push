// To run tests: "go test" or "ginkgo --succinct --slowSpecThreshold=10"

package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	body         []byte
	err          error
	params       martini.Params
	r            martini.Router
	response     *httptest.ResponseRecorder
    spicoli_env  string
)

// Before anything, do this.
var _ = BeforeSuite(func() {
	spicoli_env = "qa"

	// Startup a concurrent process to handle various system
	// events during execution.  Typically these are for notifications
	// and any other things that need to be dispatched.
    go func() {
		for {
			select {
			case <-GetInboundNotifier():
				DepleteInboundList()
			case <-GetOutboundNotifier():
				DepleteOutboundList()
			case <-GetFlushTicker():
				FlushSlackers()
				FlushRequests()
				LoadSlackers()
				LoadRequests()
			}
		}
	}()

}) // BeforeSuite

func init() {
}


func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spicoli Application API Suite")
} // func

func GetSession() {
	// Login and get a session if there is not one.
} // func

func SetAdminHeader(method string, route string, body io.Reader) *http.Request {
	GetSession()
	request, _ := http.NewRequest(method, route, body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("SPICOLI-ADMIN", "2459df92-364f-472b-975c-4fb8cc1cce54")
	return request
}

// Standard route requests.  Try to use these whenever possible.
func RequestNoAuth(method string, route string, handler martini.Handler, params martini.Params) *httptest.ResponseRecorder {
	r.Get(route, handler)
	request, _ := http.NewRequest(method, route, nil)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	m.ServeHTTP(response, request)
	return response
} // func

//
func PostRequestNoAuth(method string, route string, handler martini.Handler, body io.Reader, params martini.Params, skeleton interface{}) *httptest.ResponseRecorder {
	r.Post(route, binding.Json(skeleton), handler)
	request, _ := http.NewRequest(method, route, body)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
	return response
} // func

//
func PutRequestNoAuth(method string, route string, handler martini.Handler, body io.Reader, params martini.Params, skeleton interface{}) *httptest.ResponseRecorder {
	r.Put(route, binding.Json(skeleton), handler)
	request, _ := http.NewRequest(method, route, body)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
	return response
} // func

//
func DeleteRequestNoAuth(method string, route string, handler martini.Handler, params martini.Params) *httptest.ResponseRecorder {
	r.Delete(route, handler)
	request, _ := http.NewRequest(method, route, nil)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	m.ServeHTTP(response, request)
	return response
} // func

// After everything, do this.
var _ = AfterSuite(func() {

})
