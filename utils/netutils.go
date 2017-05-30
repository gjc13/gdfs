package utils

import (
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/proxy"

	"github.com/golang/glog"
)

func SetupProxyFromEnv() {
	envProxy := os.Getenv("https_proxy")
	if proxyUrl, _ := url.Parse(envProxy); proxyUrl != nil {
		if dialer, err := proxy.FromURL(proxyUrl, proxy.Direct); err == nil {
			glog.Infof("Setting proxy to %s\n", proxyUrl)
			http.DefaultTransport = &http.Transport{Dial: dialer.Dial}
		}
	}
}
