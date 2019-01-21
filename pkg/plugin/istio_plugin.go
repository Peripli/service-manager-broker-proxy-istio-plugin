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

	peripliContext := &PeripliContext{request: request, next: next}
	client := router.InterceptedOsbClient{&router.OsbClient{peripliContext}, i.interceptor}
	bindId := extractBindId(request.URL.Path)

	bindResponse, err := client.Bind(bindId, &bindRequest)

	return peripliContext.JSON(bindResponse, err)
}

func (i *IstioPlugin) Unbind(request *web.Request, next web.Handler) (*web.Response, error) {
	log.Printf("IstioPlugin unbind was triggered\n")
	peripliContext := &PeripliContext{request: request, next: next}
	client := router.InterceptedOsbClient{&router.OsbClient{peripliContext}, i.interceptor}
	bindId := extractBindId(request.URL.Path)
	err := client.Unbind(bindId)
	return peripliContext.JSON(nil, err)
}

func (i *IstioPlugin) FetchCatalog(request *web.Request, next web.Handler) (*web.Response, error) {
	peripliContext := &PeripliContext{request: request, next: next}
	client := router.InterceptedOsbClient{&router.OsbClient{peripliContext}, i.interceptor}

	catalog, err := client.GetCatalog()

	return peripliContext.JSON(catalog, err)
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
	config.BindEnv("network_profile")
	config.SetDefault("service_name_prefix", "istio-")
	consumerInterceptor := router.ConsumerInterceptor{}
	consumerInterceptor.ServiceNamePrefix = config.GetString("service_name_prefix")
	consumerInterceptor.NetworkProfile = config.GetString("network_profile")
	consumerInterceptor.ConsumerId = config.GetString("consumer_id")
	log.Printf("IstioPlugin starting with configuration service_name_prefix=%s consumer_id=%s network_profile=%s\n",
		consumerInterceptor.ServiceNamePrefix, consumerInterceptor.ConsumerId, consumerInterceptor.NetworkProfile)
	consumerInterceptor.ConfigStore = configStore
	return consumerInterceptor
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

func InitIstioPlugin(api *web.API) {
	istioPlugin := &IstioPlugin{interceptor: createConsumerInterceptor(router.NewInClusterConfigStore())}
	api.RegisterPlugins(istioPlugin)
}
