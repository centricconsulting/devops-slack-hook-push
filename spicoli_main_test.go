package main

import (
	"bytes"
	"encoding/json"
    "log"
	"github.com/go-martini/martini"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Main", func() {

	var (
		params martini.Params
	)

	BeforeEach(func() {
		r = martini.NewRouter()
	}) // BeforeEach

	// Look at the add new container actions.
	Context("Add New Module", func() {
		BeforeEach(func() {
			mod = julie.Module{}
			api = julie.ApiModule{}
			apis = julie.ApiModules{}
			params = make(map[string]string)
		}) // BeforeEach

		// Going in completely cold should give us an unauthorized error.
		It("POST '/module' will return a 401 status code", func() {
			rsp := PostRequestNoAuth("POST", "/module", julie.AddModule, bytes.NewReader(body), params, mod)
			Expect(rsp.Code).To(Equal(http.StatusUnauthorized), "When no token is passed, it should be an immediate rejection.")
		}) // It

		// Get a token from the system to give us authority. But there should be no data unless we are not in the unit test env.
		It("POST '/module' will return a 501 status code", func() {
			body, err = json.Marshal(mod)
			rsp := PostRequestWithAuth("POST", "/module", julie.AddModule, bytes.NewReader(body), params, mod)
			Expect(rsp.Code).To(Equal(http.StatusNotImplemented))
		}) // It
	}) // Context

}) // Describe
