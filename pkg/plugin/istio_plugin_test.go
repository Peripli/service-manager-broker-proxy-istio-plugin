package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Peripli/istio-broker-proxy/pkg/router"



	"github.com/Peripli/istio-broker-proxy/pkg/model"
	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"
)

func TestIstioPluginRegistration(t *testing.T) {
	g := NewGomegaWithT(t)

	api := web.API{}
	istioPlugin := &IstioPlugin{}
	api.RegisterPlugins(istioPlugin)
	g.Expect(len(api.Filters)).To(Equal(2))
}

func TestIstioPluginBind(t *testing.T) {
		g := NewGomegaWithT(t)
	var err error
	interceptor := SpyPostBindInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{}
	sourceEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	nextHandler.bindResponseBody, _ = json.Marshal(model.BindResponse{Endpoints: []model.Endpoint{sourceEndpoint}})
	nextHandler.adaptResponseBody, _ = json.Marshal(model.BindResponse{Endpoints: []model.Endpoint{targetEndpoint}})

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	expectedBindRequest := model.BindRequest{
		NetworkData:          model.NetworkDataRequest{NetworkProfileId: "test"},
		AdditionalProperties: map[string]json.RawMessage{}}
	request := web.Request{Request: &origRequest}
	request.Body, err = json.Marshal(expectedBindRequest)
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())

	var bindRequest model.BindRequest
	err = json.Unmarshal(nextHandler.bindRequestBody, &bindRequest)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bindRequest).To(Equal(expectedBindRequest))

	var bindResponse model.BindResponse
	err = json.Unmarshal(response.Body, &bindResponse)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(interceptor.bindId).To(Equal("34234234234-43535-345345345"))

}

func TestIstioPluginBindForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{bindStatusCode: http.StatusForbidden}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	expectedBindRequest := model.BindRequest{
		NetworkData:          model.NetworkDataRequest{NetworkProfileId: "test"},
		AdditionalProperties: map[string]json.RawMessage{}}
	request := web.Request{Request: &origRequest}
	request.Body, err = json.Marshal(expectedBindRequest)
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusForbidden))

}

func TestIstioPluginBindInvalidInput(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("sdfsf")}
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusBadRequest))

}

func TestIstioPluginBindHandleFails(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{err: errors.New("oops")}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}
	g.Expect(err).NotTo(HaveOccurred())

	_, err = plugin.Bind(&request, &nextHandler)

	g.Expect(err).To(HaveOccurred())

}

func TestIstioPluginBindInvalidBindResponse(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}
	g.Expect(err).NotTo(HaveOccurred())

	_, err = plugin.Bind(&request, &nextHandler)

	g.Expect(err).To(HaveOccurred())

}

func TestIstioPluginBindOkButAdaptForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.ConsumerInterceptor{}}
	nextHandler := SpyWebHandler{adaptStatusCode: http.StatusForbidden, bindResponseBody: []byte("{}")}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusForbidden))

}

func TestIstioPluginBindInvalidAdaptCredentialsResponse(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.ConsumerInterceptor{}}
	nextHandler := SpyWebHandler{bindResponseBody: []byte("{}")}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}
	g.Expect(err).NotTo(HaveOccurred())

	_, err = plugin.Bind(&request, &nextHandler)

	g.Expect(err).To(HaveOccurred())

}

func TestIstioPluginAdaptCredentials(t *testing.T) {
	g := NewGomegaWithT(t)

	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{}

	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	nextHandler.adaptResponseBody, _ = json.Marshal(model.BindResponse{Endpoints: []model.Endpoint{targetEndpoint}})
	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	var origHeader http.Header = map[string][]string{"Authentication": {"Bearer"}}
	origRequest := http.Request{URL: origURL, Method: http.MethodPut, Header: origHeader}
	bindRequest := web.Request{Request: &origRequest}

	credentials := model.Credentials{AdditionalProperties: map[string]json.RawMessage{"password": json.RawMessage([]byte(`"abc"`))}}
	endpointMappings := []model.EndpointMapping{{Source: model.Endpoint{Host: "host", Port: 1234}, Target: targetEndpoint}}
	adaptResponse, err := plugin.AdaptCredentials(credentials, endpointMappings, &nextHandler, &bindRequest)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(nextHandler.url.Path).To(Equal("/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345/adapt_credentials"))
	g.Expect(nextHandler.method).To(Equal(http.MethodPost))
	g.Expect(nextHandler.requestHeaders).To(Equal(origHeader))

	var adaptRequest model.AdaptCredentialsRequest

	err = json.Unmarshal(nextHandler.adaptRequestBody, &adaptRequest)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(adaptRequest.Credentials).To(Equal(credentials))
	g.Expect(adaptRequest.EndpointMappings).To(Equal(endpointMappings))

	g.Expect(adaptResponse.Endpoints).To(Equal([]model.Endpoint{targetEndpoint}))

}

func TestIstioPluginAdaptCredentialsError(t *testing.T) {
	g := NewGomegaWithT(t)

	plugin := IstioPlugin{}
	nextHandler := SpyWebHandler{err: errors.New("oops")}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	bindRequest := web.Request{Request: &origRequest}

	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	credentials := model.Credentials{AdditionalProperties: map[string]json.RawMessage{"password": json.RawMessage([]byte(`"abc"`))}}
	endpointMappings := []model.EndpointMapping{{Source: model.Endpoint{Host: "host", Port: 1234}, Target: targetEndpoint}}
	_, err := plugin.AdaptCredentials(credentials, endpointMappings, &nextHandler, &bindRequest)

	g.Expect(err).To(HaveOccurred())
}

func TestIstioPluginAdaptCredentialsInvalidResponse(t *testing.T) {
	g := NewGomegaWithT(t)

	plugin := IstioPlugin{}
	nextHandler := SpyWebHandler{adaptResponseBody: []byte("dfsf")}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	bindRequest := web.Request{Request: &origRequest}

	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	credentials := model.Credentials{AdditionalProperties: map[string]json.RawMessage{"password": json.RawMessage([]byte(`"abc"`))}}
	endpointMappings := []model.EndpointMapping{{Source: model.Endpoint{Host: "host", Port: 1234}, Target: targetEndpoint}}
	_, err := plugin.AdaptCredentials(credentials, endpointMappings, &nextHandler, &bindRequest)

	g.Expect(err).To(HaveOccurred())
}

func TestIstioPluginAdaptCredentialsBadRequest(t *testing.T) {
	g := NewGomegaWithT(t)

	plugin := IstioPlugin{}
	nextHandler := SpyWebHandler{adaptResponseBody: []byte(""), adaptStatusCode: http.StatusBadRequest}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	bindRequest := web.Request{Request: &origRequest}

	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	credentials := model.Credentials{AdditionalProperties: map[string]json.RawMessage{"password": json.RawMessage([]byte(`"abc"`))}}
	endpointMappings := []model.EndpointMapping{{Source: model.Endpoint{Host: "host", Port: 1234}, Target: targetEndpoint}}
	_, err := plugin.AdaptCredentials(credentials, endpointMappings, &nextHandler, &bindRequest)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(Equal("Error during call of adapt credentials"))
	g.Expect(err.(model.HttpError).Status).To(Equal(http.StatusBadRequest))
}

func TestIstioPluginUnbind(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	interceptor := SpyPostBindInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodDelete}
	request := web.Request{Request: &origRequest}

	response, err := plugin.Unbind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusOK))
}

func TestIstioPluginUnbindWithError(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	interceptor := SpyUnbindFailingInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodDelete}
	request := web.Request{Request: &origRequest}

	_, err = plugin.Unbind(&request, &nextHandler)

	g.Expect(err).To(HaveOccurred())
}

func TestIstioPluginUnbindForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	interceptor := SpyPostBindInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{bindStatusCode: http.StatusForbidden}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodDelete}
	request := web.Request{Request: &origRequest}

	response, err := plugin.Unbind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusForbidden))
}

type SpyWebHandler struct {
	url               url.URL
	method            string
	adaptRequestBody  []byte
	adaptResponseBody []byte
	adaptStatusCode   int
	bindRequestBody   []byte
	bindResponseBody  []byte
	bindStatusCode    int
	requestHeaders    http.Header
	err               error
}

func (s *SpyWebHandler) Handle(req *web.Request) (resp *web.Response, err error) {
	s.url = *req.Request.URL
	s.method = req.Request.Method
	s.requestHeaders = req.Header
	if strings.HasSuffix(s.url.Path, "adapt_credentials") {
		if s.adaptStatusCode == 0 {
			s.adaptStatusCode = http.StatusOK
		}
		s.adaptRequestBody = req.Body
		return &web.Response{Body: s.adaptResponseBody, StatusCode: s.adaptStatusCode}, s.err
	} else {
		if s.bindStatusCode == 0 {
			s.bindStatusCode = http.StatusOK
		}
		s.bindRequestBody = req.Body
		return &web.Response{Body: s.bindResponseBody, StatusCode: s.bindStatusCode}, s.err
	}
}

type SpyPostBindInterceptor struct {
	router.NoOpInterceptor
	bindId string
}

func (s *SpyPostBindInterceptor) PostBind(request model.BindRequest, response model.BindResponse, bindingId string,
	adapt func(model.Credentials, []model.EndpointMapping) (*model.BindResponse, error)) (*model.BindResponse, error) {
	s.bindId = bindingId
	return &response, nil
}

type SpyUnbindFailingInterceptor struct {
	router.NoOpInterceptor
	bindId string
}

func (s *SpyUnbindFailingInterceptor) PostDelete(bindId string) error {
	s.bindId = bindId
	return fmt.Errorf("delete failed")
}
