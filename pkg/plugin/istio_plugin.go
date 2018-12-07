package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
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
	log.Printf("IstioPlugin bind was triggered\n")
	err := json.Unmarshal(request.Body, &bindRequest)
	if err != nil {
		return httpError(err, http.StatusBadRequest)
	}
	log.Println("execute prebind")
	bindRequest = *i.interceptor.PreBind(bindRequest)
	request.Body, _ = json.Marshal(bindRequest)

	response, err := next.Handle(request)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}

	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	var bindResponse model.BindResponse
	err = json.Unmarshal(response.Body, &bindResponse)
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
	}
	log.Println("execute postbind")
	modifiedBindResponse, err := i.interceptor.PostBind(bindRequest, bindResponse, extractBindId(request.URL.Path),
		func(credentials model.Credentials, mappings []model.EndpointMapping) (*model.BindResponse, error) {
			return i.AdaptCredentials(credentials, mappings, next, request)
		})
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
	}
	response.Body, err = json.Marshal(modifiedBindResponse)
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
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
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	if err != nil {
		logError(err)
		return nil, err
	}
	var bindResponse model.BindResponse
	err = json.Unmarshal(response.Body, &bindResponse)
	if err != nil {
		logError(err)
		return nil, err
	}
	return &bindResponse, nil
}

func httpError(err error, statusCode int) (*web.Response, error) {
	log.Printf("ERROR: %s\n", err.Error())
	httpError := model.HttpErrorFromError(err, statusCode)
	response := &web.Response{StatusCode: httpError.StatusCode}
	response.Body, err = json.Marshal(httpError)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (i *IstioPlugin) Unbind(request *web.Request, next web.Handler) (*web.Response, error) {
	log.Printf("IstioPlugin unbind was triggered\n")
	response, err := next.Handle(request)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	err = i.interceptor.PostDelete(extractBindId(request.URL.Path))
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
	}
	return response, nil
}

func (i *IstioPlugin) FetchCatalog(request *web.Request, next web.Handler) (*web.Response, error) {
	response, err := next.Handle(request)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	err = model.HttpErrorFromResponse(response.StatusCode, response.Body)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	var catalog model.Catalog
	err = json.Unmarshal(response.Body, &catalog)
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
	}
	i.interceptor.PostCatalog(&catalog)

	response.Body, err = json.Marshal(&catalog)
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
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

func createConsumerInterceptor(configStore router.ConfigStore) router.ConsumerInterceptor {
	config := viper.New()
	config.SetEnvPrefix("istio")
	config.BindEnv("service_name_prefix")
	config.BindEnv("consumer_id")
	config.SetDefault("service_name_prefix", "istio-")
	consumerInterceptor := router.ConsumerInterceptor{}
	consumerInterceptor.ServiceNamePrefix = config.GetString("service_name_prefix")
	consumerInterceptor.ConsumerId = config.GetString("consumer_id")
	log.Printf("IstioPlugin starting with configuration service_name_prefix=%s consumer_id=%s\n", consumerInterceptor.ServiceNamePrefix, consumerInterceptor.ConsumerId)
	consumerInterceptor.ConfigStore = configStore
	return consumerInterceptor
}

func InitIstioPlugin(api *web.API) {
	istioPlugin := &IstioPlugin{interceptor: createConsumerInterceptor(router.NewInClusterConfigStore())}
	api.RegisterPlugins(istioPlugin)
}
