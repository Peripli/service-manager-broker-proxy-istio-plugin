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

	peripliRestClient := &PeripliRestClient{request: request, next: next}
	client := router.InterceptedOsbClient{&router.OsbClient{peripliRestClient}, i.interceptor}
	instanceId, bindId := extractServiceIdBindId(request.URL.Path)

	bindResponse, err := client.Bind(instanceId, bindId, &bindRequest)

	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}

	peripliRestClient.response.Body, err = json.Marshal(bindResponse)
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
	}

	return peripliRestClient.response, nil
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
	peripliRestClient := &PeripliRestClient{request: request, next: next}
	client := router.InterceptedOsbClient{&router.OsbClient{peripliRestClient}, i.interceptor}
	instanceId, bindId := extractServiceIdBindId(request.URL.Path)
	err := client.Unbind(instanceId, bindId)
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	return peripliRestClient.response, nil
}

func (i *IstioPlugin) FetchCatalog(request *web.Request, next web.Handler) (*web.Response, error) {
	peripliRestClient := &PeripliRestClient{request: request, next: next}
	client := router.InterceptedOsbClient{&router.OsbClient{peripliRestClient}, i.interceptor}

	catalog, err := client.GetCatalog()
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}

	peripliRestClient.response.Body, err = json.Marshal(&catalog)
	if err != nil {
		return httpError(err, http.StatusInternalServerError)
	}

	return peripliRestClient.response, nil
}

func extractServiceIdBindId(path string) (string, string) {
	splitPath := strings.Split(path, "/")
	if splitPath[len(splitPath)-2] != "service_bindings" {
		panic(fmt.Sprintf("Failed to extract binding id from path %s", path))
	}
	return splitPath[len(splitPath)-3], splitPath[len(splitPath)-1]
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
