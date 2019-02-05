package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	g.Expect(len(api.Filters)).To(Equal(3))
}

func TestIstioPluginBind(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	interceptor := SpyPostBindInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{}
	sourceEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	nextHandler.responseBody, _ = json.Marshal(model.BindResponse{Endpoints: []model.Endpoint{sourceEndpoint}})
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
	err = json.Unmarshal(nextHandler.requestBody, &bindRequest)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bindRequest).To(Equal(expectedBindRequest))

	var bindResponse model.BindResponse
	err = json.Unmarshal(response.Body, &bindResponse)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(interceptor.bindId).To(Equal("34234234234-43535-345345345-xxx"))

}

func TestIstioPluginBindForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{statusCode: http.StatusForbidden}

	origURL, _ := url.Parse("http://host:80/some/other/path/that/we/dont/controll/at/all/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
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

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(*model.HttpError).ErrorMsg).To(Equal("oops"))

}

func TestIstioPluginBindInvalidBindResponse(t *testing.T) {
	g := NewGomegaWithT(t)
	plugin := IstioPlugin{interceptor: router.NoOpInterceptor{}}
	nextHandler := SpyWebHandler{}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}

	response, err := plugin.Bind(&request, &nextHandler)
	g.Expect(err).NotTo(HaveOccurred())
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Can't unmarshal response from"))
}

func TestIstioPluginBindOkButAdaptForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	plugin := IstioPlugin{interceptor: router.ConsumerInterceptor{NetworkProfile: "urn:local.test:public"}}
	nextHandler := SpyWebHandler{statusCode: http.StatusForbidden, responseBody: []byte("{}")}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusForbidden))

}

func TestIstioPluginBindInvalidAdaptCredentialsResponseWithoutEndpoints(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	configStore := &router.MockConfigStore{}
	plugin := IstioPlugin{interceptor: router.ConsumerInterceptor{ConfigStore: configStore, NetworkProfile: "urn:local.test:public"}}
	nextHandler := SpyWebHandler{responseBody: []byte(`{"network_data": {"network_profile_id": "urn:local.test:public"}}`)}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte(`{}`)}
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Can't unmarshal response from"))

}

func TestIstioPluginBindInvalidAdaptCredentialsResponseWithEndpoints(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	configStore := &router.MockConfigStore{}
	plugin := IstioPlugin{interceptor: router.ConsumerInterceptor{ConfigStore: configStore, NetworkProfile: "urn:local.test:public"}}
	targetEndpoint := model.Endpoint{Host: "host2", Port: 8888}
	endpointsResponse, _ := json.Marshal(model.BindResponse{Endpoints: []model.Endpoint{targetEndpoint},
		NetworkData: model.NetworkDataResponse{
			NetworkProfileId: "urn:local.test:public",
			Data:             model.DataResponse{Endpoints: []model.Endpoint{targetEndpoint}}}})
	nextHandler := SpyWebHandler{responseBody: endpointsResponse}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodPut}
	request := web.Request{Request: &origRequest, Body: []byte("{}")}
	g.Expect(err).NotTo(HaveOccurred())

	response, err := plugin.Bind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("Can't unmarshal response from"))

	g.Expect(configStore.CreatedServices).To(HaveLen(0))
	g.Expect(configStore.CreatedIstioConfigs).To(HaveLen(0))
	g.Expect(configStore.DeletedServices).To(HaveLen(1))
	g.Expect(configStore.DeletedIstioConfigs).To(HaveLen(6))

}

func TestIstioPluginUnbind(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	interceptor := SpyPostBindInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345?parameter=true")
	origRequest := http.Request{URL: origURL, Method: http.MethodDelete}
	request := web.Request{Request: &origRequest}

	response, err := plugin.Unbind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusOK))
	g.Expect(nextHandler.url.RawQuery).To(Equal("parameter=true"))
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

	response, err := plugin.Unbind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.(*model.HttpError).ErrorMsg).To(Equal("delete failed"))
}

func TestIstioPluginUnbindForbidden(t *testing.T) {
	g := NewGomegaWithT(t)
	var err error
	interceptor := SpyPostBindInterceptor{}
	plugin := IstioPlugin{interceptor: &interceptor}
	nextHandler := SpyWebHandler{statusCode: http.StatusForbidden}

	origURL, _ := url.Parse("http://host:80/v2/service_instances/3234234-234234-234234/service_bindings/34234234234-43535-345345345")
	origRequest := http.Request{URL: origURL, Method: http.MethodDelete}
	request := web.Request{Request: &origRequest}

	response, err := plugin.Unbind(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(response.StatusCode).To(Equal(http.StatusForbidden))
}

func TestIstioPluginFetchCatalog(t *testing.T) {
	g := NewGomegaWithT(t)

	interceptor := router.ConsumerInterceptor{ServiceNamePrefix: "istio-"}
	plugin := IstioPlugin{interceptor: &interceptor}
	catalog := model.Catalog{[]model.Service{{Name: "istio-servicename"}}}

	origURL, _ := url.Parse("http://host:80/v2/catalog")
	origRequest := http.Request{URL: origURL, Method: http.MethodGet}

	catalogBody, _ := json.Marshal(&catalog)
	nextHandler := SpyWebHandler{statusCode: http.StatusOK, responseBody: catalogBody}

	var pathParams map[string]string
	request := web.Request{Request: &origRequest, PathParams: pathParams}

	response, err := plugin.FetchCatalog(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(response.Body)).To(ContainSubstring("servicename"))
	g.Expect(string(response.Body)).NotTo(ContainSubstring("istio-"))
}

func TestFailingFetchCatalog(t *testing.T) {
	g := NewGomegaWithT(t)

	interceptor := router.ConsumerInterceptor{ServiceNamePrefix: "istio-"}
	plugin := IstioPlugin{interceptor: &interceptor}

	origURL, _ := url.Parse("http://host:80/v2/catalog")
	origRequest := http.Request{URL: origURL, Method: http.MethodGet}

	someError := fmt.Errorf("some problem")
	nextHandler := SpyWebHandler{err: someError}

	var pathParams map[string]string
	request := web.Request{Request: &origRequest, PathParams: pathParams}

	response, err := plugin.FetchCatalog(&request, &nextHandler)

	g.Expect(err).NotTo(HaveOccurred())
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(Equal("some problem"))
}

func TestCreateConsumerInterceptor(t *testing.T) {
	g := NewGomegaWithT(t)
	os.Setenv("ISTIO_SERVICE_NAME_PREFIX", "hello-")
	os.Setenv("ISTIO_CONSUMER_ID", "myconsumer-id")
	ci := createConsumerInterceptor(nil)
	g.Expect(ci.ConsumerId).To(Equal("myconsumer-id"))
	g.Expect(ci.ServiceNamePrefix).To(Equal("hello-"))

}

type SpyWebHandler struct {
	url               url.URL
	method            string
	adaptResponseBody []byte
	requestBody       []byte
	responseBody      []byte
	statusCode        int
	requestHeaders    http.Header
	err               error
}

func (s *SpyWebHandler) Handle(req *web.Request) (resp *web.Response, err error) {
	s.url = *req.Request.URL
	s.method = req.Request.Method
	s.requestHeaders = req.Header
	if s.statusCode == 0 {
		s.statusCode = http.StatusOK
	}
	s.requestBody = req.Body
	var responseBody []byte
	if strings.HasSuffix(s.url.Path, "adapt_credentials") {
		responseBody = s.adaptResponseBody
	} else {
		responseBody = s.responseBody
	}
	return &web.Response{Body: responseBody, StatusCode: s.statusCode}, s.err
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
