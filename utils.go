package main

import (
	"net/http"
	"strconv"
)

func getPriority(req *http.Request) int64 {
	if values, hasProperty := req.Header["X-Priority"]; hasProperty {
		if priority, parseError := strconv.ParseInt(values[0], 10, 64); parseError != nil {
			return priority
		}
	}
	requestURI := req.RequestURI
	if weight, ok := requestPriority[requestURI]; ok {
		return weight
	} else {
		return 0
	}
}

func getWeight(req *http.Request) int64 {
	if values, hasProperty := req.Header["X-Weight"]; hasProperty {
		if weight, parseError := strconv.ParseInt(values[0], 10, 64); parseError != nil {
			return weight
		}
	}
	requestURI := req.RequestURI
	if weight, ok := requestWeight[requestURI]; ok {
		return weight
	} else {
		return 0
	}
}
