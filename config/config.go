package config

// MaxResources 最大资源数
var MaxResources int64 = 24

// RequestResources 单次需要的资源
var RequestResources = map[string]int64{
	"/test1": 24,
	"/test2": 2,
}

// RequestPriority 请求优先级
var RequestPriority = map[string]int64{
	"/test1": 1,
	"/test2": 20,
}

var ForwardURLPrefix = "http://localhost:8000"

func init() {
	// TODO load config
}
