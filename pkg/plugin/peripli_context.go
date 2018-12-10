package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/Peripli/istio-broker-proxy/pkg/model"
	"github.com/Peripli/istio-broker-proxy/pkg/router"
	"github.com/Peripli/service-manager/pkg/web"
	"log"
	"net/http"
)

type PeripliContext struct {
	request  *web.Request
	next     web.Handler
	response *web.Response
	err      error
}

type PeripliRestRequest struct {
	*PeripliContext
}

type PeripliRestResponse struct {
	*PeripliContext
}

func (client *PeripliContext) Get() router.RestRequest {
	client.request.Method = http.MethodGet
	return &PeripliRestRequest{client}
}

func (client *PeripliContext) Delete() router.RestRequest {
	client.request.Method = http.MethodDelete
	return &PeripliRestRequest{client}
}

func (client *PeripliContext) Post(request interface{}) router.RestRequest {
	client.request.Method = http.MethodPost
	client.request.Body, client.err = json.Marshal(request)
	return &PeripliRestRequest{client}
}

func (client *PeripliContext) Put(request interface{}) router.RestRequest {
	client.request.Method = http.MethodPut
	client.request.Body, client.err = json.Marshal(request)
	return &PeripliRestRequest{client}
}

func (r *PeripliRestRequest) Path(path string) router.RestRequest {
	r.request.URL.Path = path
	return r
}

func (r *PeripliRestRequest) Do() router.RestResponse {
	response := PeripliRestResponse{r.PeripliContext}
	if r.err != nil {
		return &response
	}

	response.response, response.err = r.next.Handle(r.request)
	if response.err != nil {
		return &response
	}

	response.err = model.HttpErrorFromResponse(response.response.StatusCode, response.response.Body)
	return &response

}

func (o *PeripliRestResponse) Into(result interface{}) error {
	if o.err != nil {
		return o.err
	}
	o.err = json.Unmarshal(o.response.Body, result)

	if nil != o.err {
		o.err = fmt.Errorf("Can't unmarshal response from %s: %s", o.request.URL.String(), o.err.Error())
		log.Printf("ERROR: %s\n", o.err.Error())
		return o.err
	}
	return nil
}

func (o *PeripliRestResponse) Error() error {
	return o.err
}

func (o *PeripliContext) JSON(result interface{}, err error) (*web.Response, error) {
	if err != nil {
		return httpError(err, http.StatusBadGateway)
	}
	if result != nil {
		o.response.Body, err = json.Marshal(result)
		if err != nil {
			return httpError(err, http.StatusInternalServerError)
		}
	}
	return o.response, nil
}
