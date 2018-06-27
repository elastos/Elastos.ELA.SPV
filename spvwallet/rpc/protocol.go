package rpc

import (
	"fmt"

	"github.com/elastos/Elastos.ELA.SPV/spvwallet/config"
)

var (
	RPCPort = config.Values().RPCPort
	RPCAddr = fmt.Sprint("http://127.0.0.1:", RPCPort, "/spvwallet/")
)

type Req struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

type Resp struct {
	Code   int         `json:"code"`
	Result interface{} `json:"result"`
}

var (
	MarshalRequestError    = Resp{301, "MarshalRequestError"}
	PostRequestError       = Resp{302, "PostRequestError"}
	ReadResponseError      = Resp{303, "ReadResponseError"}
	UnmarshalResponseError = Resp{304, "UnmarshalResponseError"}
)

var (
	NonPostRequest        = Resp{401, "NonPostRequest"}
	EmptyRequestBody      = Resp{402, "EmptyRequestBody"}
	ReadRequestError      = Resp{403, "ReadRequestError"}
	UnmarshalRequestError = Resp{404, "UnmarshalRequestError"}
	InvalidMethod         = Resp{405, "InvalidMethod"}
	InvalidParameter      = Resp{406, "InvalidParameter"}
)

func Success(result interface{}) Resp {
	return Resp{0, result}
}

func FunctionError(error string) Resp {
	return Resp{407, error}
}
