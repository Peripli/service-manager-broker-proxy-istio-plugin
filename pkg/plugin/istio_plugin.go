package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Peripli/istio-broker-proxy/pkg/model"
	"github.com/Peripli/istio-broker-proxy/pkg/router"
	"github.com/Peripli/service-manager/pkg/web"
)

type IstioPlugin struct {
	interceptor router.ServiceBrokerInterceptor
}

func (i *IstioPlugin) Name() string {
	return "istio"
}

func logError(err error) {
	log.Printf("error occured: %s\n", err.Error())
}

func (i *IstioPlugin) Bind(request *web.Request, next web.Handler) (*web.Response, error) {
	var bindRequest model.BindRequest
	log.Printf("IstioPlugin bind was triggered with request adaptRequestBody: %s\n", string(request.Body))
	err := json.Unmarshal(request.Body, &bindRequest)
	if err != nil {
		logError(err)
		return &web.Response{StatusCode: http.StatusBadRequest}, nil
	}
	log.Println("execute prebind")
	bindRequest = *i.interceptor.PreBind(bindRequest)
	request.Body, _ = json.Marshal(bindRequest)

	response, err := next.Handle(request)
	if err != nil {
		logError(err)
		return nil, err
	}

	if response.StatusCode/100 != 2 {
		logError(fmt.Errorf("http response code %d", response.StatusCode))
		return response, nil
	}
	var bindResponse model.BindResponse
	err = json.Unmarshal(response.Body, &bindResponse)
	if err != nil {
		logError(err)
		return nil, err
	}
	log.Println("execute postbind")
	modifiedBindResponse, err := i.interceptor.PostBind(bindRequest, bindResponse, extractBindId(request.URL.Path),
		func(credentials model.Credentials, mappings []model.EndpointMapping) (*model.BindResponse, error) {
			return i.AdaptCredentials(credentials, mappings, next, request)
		})
	if err != nil {
		logError(err)
		httpError, ok := err.(model.HttpError)
		if ok {
			return &web.Response{StatusCode: httpError.Status}, nil
		}
		return nil, err
	}
	response.Body, err = json.Marshal(modifiedBindResponse)
	if err != nil {
		logError(err)
		return nil, err
	}
	return response, nil
}

func (i *IstioPlugin) AdaptCredentials(credentials model.Credentials, endpointMappings []model.EndpointMapping, next web.Handler, bindRequest *web.Request) (*model.BindResponse, error) {
	request := web.Request{Request: bindRequest.Request, PathParams: bindRequest.PathParams}
	request.URL.Path = request.URL.Path + "/adapt_credentials"
	request.Method = http.MethodPost
	adaptCredentialsRequest := model.AdaptCredentialsRequest{credentials, endpointMappings}
	var err error
	request.Body, err = json.Marshal(&adaptCredentialsRequest)
	if err != nil {
		logError(err)
		return nil, err
	}
	response, err := next.Handle(&request)
	if err != nil {
		logError(err)
		return nil, err
	}
	if response.StatusCode/100 != 2 {
		httpError := model.HttpError{Status: response.StatusCode, Message: fmt.Sprintf("Error during call of adapt credentials")}
		logError(httpError)
		return nil, httpError
	}
	var bindResponse model.BindResponse
	err = json.Unmarshal(response.Body, &bindResponse)
	if err != nil {
		logError(err)
		return nil, err
	}
	return &bindResponse, nil
}

func (i *IstioPlugin) Unbind(request *web.Request, next web.Handler) (*web.Response, error) {
	log.Printf("IstioPlugin unbind was triggered with request adaptRequestBody: %s\n", string(request.Body))
	err := i.interceptor.PostDelete(extractBindId(request.URL.Path))
	if err != nil {
		return nil, err
	}
	return next.Handle(request)
}

func (i *IstioPlugin) FetchCatalog(request *web.Request, next web.Handler) (*web.Response, error) {
	response, err := next.Handle(request)
	if err != nil {
		return response, err
	}

	var catalog model.Catalog
	err = json.Unmarshal(response.Body, &catalog)
	if err != nil{
		logError(err)
		return nil, err
	}
	i.interceptor.PostCatalog(&catalog)

	response.Body, err = json.Marshal(&catalog)
	if err != nil{
		logError(err)
		return nil, err
	}

	return response, nil
}

func extractBindId(path string) string {
	splitPath := strings.Split(path, "/")
	if splitPath[len(splitPath)-2] != "service_bindings" {
		panic(fmt.Sprintf("Failed to extract binding id from path %s", path))
	}
	return splitPath[len(splitPath)-1]
}

func createConsumerInterceptor() router.ConsumerInterceptor {
	consumerInterceptor := router.ConsumerInterceptor{}
	consumerInterceptor.ServiceNamePrefix = "istio-"
	consumerInterceptor.ConsumerId = "client.istio.sapcloud.io"
	consumerInterceptor.ConfigStore = router.NewInClusterConfigStore()
	return consumerInterceptor
}

func InitIstioPlugin(api *web.API) {
	istioPlugin := &IstioPlugin{interceptor: createConsumerInterceptor()}
	api.RegisterPlugins(istioPlugin)
}
