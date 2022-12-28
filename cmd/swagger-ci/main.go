package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math/rand"
	netHttp "net/http"
	"net/url"
	"strings"

	"github.com/bxcodec/faker/v4"

	swaggerCi "github.com/thoohv5/swagger-ci"
	"github.com/thoohv5/swagger-ci/util/http"
)

var swaggerUrl string

func init() {
	flag.StringVar(&swaggerUrl, "url", "https://127.0.0.1/swagger/doc.json", "swagger json")
}

func main() {
	flag.Parse()

	// 解析swagger文件
	swagger := new(swaggerCi.Swagger)
	if err := http.Get(context.Background(), swaggerUrl, swagger, http.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})); err != nil {
		fmt.Println("swagger url err", err)
		return
	}

	// 拼接url
	urlParam, err := url.Parse(swaggerUrl)
	if err != nil {
		return
	}
	host := fmt.Sprintf("%s://%s", urlParam.Scheme, urlParam.Host)

	// 解析数据
	for url, path := range swagger.Paths {
		for method, request := range path {
			param := make(map[string]interface{})
			for _, parameter := range request.Parameters {
				var value interface{}
				pName := parameter.Name

				if parameter.In == "body" {
					model := strings.TrimPrefix(parameter.Schema.Ref, "#/definitions/")
					for name, p := range swagger.Definitions[model].Properties {
						pName = name
						if df := p.Default; df != nil {
							value = df
						} else if enum := p.Enum; len(enum) > 0 {
							value = enum[rand.Intn(len(enum))]
						} else {
							switch p.Type {
							case "string":
								value = faker.Word()
							case "boolean":
								value = true
							case "integer":
								value = faker.UnixTime()
							case "array":
							case "object":

							}

						}
						param[pName] = value
					}
					continue
				}

				if df := parameter.Default; df != nil {
					value = df
				} else if enum := parameter.Enum; len(enum) > 0 {
					value = enum[rand.Intn(len(enum))]
				} else {
					switch parameter.Type {
					case "string":
						value = faker.Word()
					case "boolean":
						value = true
					case "integer":
						value = faker.UnixTime()
					case "array":
					case "object":

					}
				}

				if parameter.In == "path" {
					url = strings.ReplaceAll(url, fmt.Sprintf("{%s}", parameter.Name), fmt.Sprint(value))
					continue
				}

				param[pName] = value
			}

			resp := new(netHttp.Response)
			ret := make(map[string]interface{})
			switch method {
			case "get":
				if err = http.Get(context.Background(), fmt.Sprintf("%s%s", host, url), &ret, http.WithParam(param), http.WithTLSClientConfig(&tls.Config{
					InsecureSkipVerify: true,
				}), http.WithResponse(resp), http.WithLog(&defaultLog{})); err != nil {
					panic(err)
				}
			case "post":
				if err = http.Post(context.Background(), fmt.Sprintf("%s%s", host, url), param, &ret, http.WithTLSClientConfig(&tls.Config{
					InsecureSkipVerify: true,
				}), http.WithResponse(resp), http.WithLog(&defaultLog{})); err != nil {
					panic(err)
				}
			case "put":
				if err = http.Put(context.Background(), fmt.Sprintf("%s%s", host, url), param, &ret, http.WithTLSClientConfig(&tls.Config{
					InsecureSkipVerify: true,
				}), http.WithResponse(resp), http.WithLog(&defaultLog{})); err != nil {
					panic(err)
				}
			case "delete":
				if err = http.Delete(context.Background(), fmt.Sprintf("%s%s", host, url), &ret, http.WithParam(param), http.WithTLSClientConfig(&tls.Config{
					InsecureSkipVerify: true,
				}), http.WithResponse(resp), http.WithLog(&defaultLog{})); err != nil {
					panic(err)
				}
			}

			if resp.StatusCode == netHttp.StatusOK || resp.StatusCode == netHttp.StatusBadRequest && (ret["code"] == "10010100003" || ret["code"] == "10010100002") {
				fmt.Printf("method: %s, ret:%v, resp:%v\n", method, ret, resp.StatusCode)
			} else {
				panic(fmt.Sprintf("\033[0;31m method: %s, ret:%v, resp:%v\n", method, ret, resp.StatusCode))
			}
		}
	}

}

type defaultLog struct {
}

func (dl *defaultLog) Println(a ...interface{}) {
	fmt.Println(a...)
}
