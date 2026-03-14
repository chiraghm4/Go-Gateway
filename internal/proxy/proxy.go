package proxy

import (
	"net/url"
	"net/http/httputil"
)

func NewReverseProxy(target string) (*httputil.ReverseProxy, error) {
	parsedUrl, err := url.Parse(target)

	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedUrl)

	return proxy, nil
}