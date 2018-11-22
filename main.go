package main

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager-broker-proxy-istio-plugin/pkg/plugin"
	"unsafe"
)

func Init(api unsafe.Pointer) error {
	myApi := ((*web.API)(api))
	plugin.InitIstioPlugin(myApi)
	return nil
}
