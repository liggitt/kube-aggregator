package v1beta1

// This file contains a collection of methods that can be used from go-restful to
// generate Swagger API documentation for its models. Please read this PR for more
// information on the implementation: https://github.com/emicklei/go-restful/pull/215
//
// TODOs are ignored from the parser (e.g. TODO(andronat):... || TODO:...) if and only if
// they are on one line! For multiple line or blocks that you want to ignore use ---.
// Any context after a --- is ignored.
//
// Those methods can be generated by using hack/update-generated-swagger-docs.sh

// AUTO-GENERATED FUNCTIONS START HERE
var map_APIServer = map[string]string{
	"": "APIServer is a logical top-level container for a set of origin resources",
}

func (APIServer) SwaggerDoc() map[string]string {
	return map_APIServer
}

var map_APIServerList = map[string]string{
	"": "APIServerList is a list of APIServer objects.",
}

func (APIServerList) SwaggerDoc() map[string]string {
	return map_APIServerList
}

// AUTO-GENERATED FUNCTIONS END HERE
