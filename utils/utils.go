package utils

import (
	"github.com/fanck0605/http-schedule/config"
	"net/http"
	"strconv"
)

func GetPriority(req *http.Request) int64 {
	if values, hasProperty := req.Header["X-Priority"]; hasProperty {
		if priority, parseError := strconv.ParseInt(values[0], 10, 64); parseError != nil {
			return priority
		}
	}
	requestURI := req.RequestURI
	if weight, ok := config.RequestPriority[requestURI]; ok {
		return weight
	} else {
		return 0
	}
}

func GetWeight(req *http.Request) int64 {
	if values, hasProperty := req.Header["X-Weight"]; hasProperty {
		if weight, parseError := strconv.ParseInt(values[0], 10, 64); parseError != nil {
			return weight
		}
	}
	requestURI := req.RequestURI
	if weight, ok := config.RequestWeight[requestURI]; ok {
		return weight
	} else {
		return 0
	}
}

func GetRequestURL(req *http.Request) string {
	requestURI := req.RequestURI
	return config.ForwardURLPrefix + requestURI
}
