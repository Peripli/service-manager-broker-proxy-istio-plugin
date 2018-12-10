package plugin

import (
	"fmt"
	"github.com/Peripli/istio-broker-proxy/pkg/model"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/gomega"
	"net/http"
	"net/url"
	"testing"
)

type TestStruct struct {
	Member1 string `json:"member1"`
	Member2 int    `json:"member2"`
}

func TestPeripliContextPut(t *testing.T) {
	g := NewGomegaWithT(t)

	nextHandler := &SpyWebHandler{statusCode: http.StatusOK, responseBody: []byte(`{"member1": "string","member2": 1}`)}
	client := &PeripliContext{request: &web.Request{Request: &http.Request{URL: &url.URL{}}}, next: nextHandler}
	testStruct := TestStruct{"s", 10}
	err := client.Put(&testStruct).Do().Into(&testStruct)

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(testStruct.Member1).To(Equal("string"))
	g.Expect(testStruct.Member2).To(Equal(1))
	g.Expect(nextHandler.method).To(Equal(http.MethodPut))
	g.Expect(nextHandler.requestBody).To(MatchJSON(`{"member1": "s","member2": 10}`))

}

func TestPeripliContextWithBadRequest(t *testing.T) {
	g := NewGomegaWithT(t)

	nextHandler := &SpyWebHandler{statusCode: http.StatusBadRequest, responseBody: []byte(`{"error" : "myerror", "description" : "mydescription"}`)}
	client := &PeripliContext{request: &web.Request{Request: &http.Request{URL: &url.URL{}}}, next: nextHandler}
	err := client.Get().Do().Error()

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(*model.HttpError).StatusCode).To(Equal(http.StatusBadRequest))
	g.Expect(err.(*model.HttpError).ErrorMsg).To(Equal("myerror"))
	g.Expect(err.(*model.HttpError).Description).To(Equal("mydescription"))
	g.Expect(nextHandler.method).To(Equal(http.MethodGet))
}

func TestPeripliContextPostWithInvalidJson(t *testing.T) {
	g := NewGomegaWithT(t)

	nextHandler := &SpyWebHandler{statusCode: http.StatusOK, responseBody: []byte(``)}
	client := &PeripliContext{request: &web.Request{Request: &http.Request{URL: &url.URL{}}}, next: nextHandler}
	result := TestStruct{}
	err := client.Post(&result).Do().Into(&result)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Can't unmarshal response from"))
	g.Expect(nextHandler.method).To(Equal(http.MethodPost))
}

func TestPeripliContextDelete(t *testing.T) {
	g := NewGomegaWithT(t)

	nextHandler := &SpyWebHandler{statusCode: http.StatusOK, responseBody: []byte(`{}`)}
	client := &PeripliContext{request: &web.Request{Request: &http.Request{URL: &url.URL{}}}, next: nextHandler}
	err := client.Delete().Do().Error()

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(nextHandler.method).To(Equal(http.MethodDelete))

}

func TestPeripliContextJSON(t *testing.T) {
	g := NewGomegaWithT(t)

	client := &PeripliContext{response: &web.Response{}}
	testStruct := TestStruct{"s", 10}
	response, err := client.JSON(&testStruct, nil)

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(response.Body).To(MatchJSON(`{"member1": "s","member2": 10}`))
}

func TestPeripliContextJSONWithError(t *testing.T) {
	g := NewGomegaWithT(t)

	client := &PeripliContext{response: &web.Response{}}
	response, err := client.JSON(nil, fmt.Errorf("Test"))

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(response.Body).To(MatchJSON(`{"error": "Test","description": ""}`))
	g.Expect(response.StatusCode).To(Equal(http.StatusBadGateway))
}
