package config

// MaxWeight 最大资源数的权重
var MaxWeight int64 = 24

// RequestWeight 单词请求需要的权重
var RequestWeight = map[string]int64{
	"/test1": 24,
	"/test2": 2,
}

var RequestPriority = map[string]int64{
	"/test1": 1,
	"/test2": 20,
}

var ForwardURLPrefix = "http://localhost:8000"

func init() {
	// TODO load config
}
