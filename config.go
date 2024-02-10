package main

var maxResources int64 = 24

var requestWeight = map[string]int64{
	"/test1": 24,
	"/test2": 2,
}

var requestPriority = map[string]int64{
	"/test1": 1,
	"/test2": 20,
}

func init() {
	// TODO load config
}
