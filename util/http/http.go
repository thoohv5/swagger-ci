package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	nethttp "net/http"
	neturl "net/url"
	"time"
)

type (
	IHttp interface {
		// Get get
		Get(ctx context.Context, url string, result interface{}, opts ...Option) error
		// Post post
		Post(ctx context.Context, url string, param map[string]interface{}, result interface{}, opts ...Option) error
		// Put put
		Put(ctx context.Context, url string, param map[string]interface{}, result interface{}, opts ...Option) error
		// Delete delete
		Delete(ctx context.Context, url string, result interface{}, opts ...Option) error
	}
	// Parameter 参数
	Parameter struct {
		// url
		url string
		// 请求方式
		method Method
		// 超时时间
		timeout time.Duration
		// header
		header map[string]string
		// param
		param map[string]interface{}
		// preDeal
		preDeal []func(r *Parameter) error
		// reader
		body io.Reader
		// tls
		tLSClientConfig *tls.Config
		// log
		log ILogger

		// 返回值
		response *nethttp.Response
	}
	Option interface {
		apply(*Parameter)
	}
	optionFunc func(*Parameter)
	http       struct {
		*Parameter
	}
	ILogger interface {
		Println(a ...interface{})
	}
)

func (f optionFunc) apply(o *Parameter) {
	f(o)
}

func (p *Parameter) SetBody(body io.Reader) {
	p.body = body
}

type Method string

const (
	DefaultTimeOut        = 3 * time.Second
	MethodPost     Method = "POST"
	MethodGet      Method = "GET"
	MethodPut      Method = "PUT"
	MethodDelete   Method = "DELETE"
)

func NewHttp() IHttp {
	h := &http{
		Parameter: &Parameter{
			method:  MethodGet,
			timeout: DefaultTimeOut,
			header: map[string]string{
				"Content-Type": "application/json",
			},
			param: map[string]interface{}{},
			tLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}
	return h
}

func (r *http) withOpt(opts ...Option) error {
	for _, o := range opts {
		o.apply(r.Parameter)
	}
	return nil
}

func WithUrl(url string) Option {
	return optionFunc(func(r *Parameter) {
		r.url = url
	})
}

func WithMethod(method Method) Option {
	return optionFunc(func(r *Parameter) {
		r.method = method
	})
}

func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(r *Parameter) {
		r.timeout = timeout * time.Second
	})
}

func WithParam(params map[string]interface{}) Option {
	return optionFunc(func(r *Parameter) {
		for key, val := range params {
			r.param[key] = val
		}
	})
}

func WithHeader(headers map[string]string) Option {
	return optionFunc(func(r *Parameter) {
		for key, val := range headers {
			r.header[key] = val
		}
	})
}

func WithPreDeal(preDeal func(r *Parameter) error) Option {
	return optionFunc(func(r *Parameter) {
		r.preDeal = append(r.preDeal, preDeal)
	})
}

func WithTLSClientConfig(tLSClientConfig *tls.Config) Option {
	return optionFunc(func(r *Parameter) {
		r.tLSClientConfig = tLSClientConfig
	})
}

func WithResponse(response *nethttp.Response) Option {
	return optionFunc(func(r *Parameter) {
		r.response = response
	})
}

func WithLog(log ILogger) Option {
	return optionFunc(func(r *Parameter) {
		r.log = log
	})
}

func (r *http) request(ctx context.Context, url string, result interface{}, opts ...Option) (err error) {

	opts = append([]Option{WithUrl(url)}, opts...)
	// 可选参数
	if err = r.withOpt(opts...); nil != err {
		return fmt.Errorf("request withOpt err, opts: %v, %w", opts, err)
	}

	// 预处理
	for _, preDeal := range r.preDeal {
		if err := preDeal(r.Parameter); nil != err {
			return fmt.Errorf("request callback err, r: %v, %w", r, err)
		}
	}

	// 组装request
	req, err := nethttp.NewRequestWithContext(ctx, string(r.method), r.url, r.body)
	if nil != err {
		return fmt.Errorf("request NewRequestWithContext err, url: %s, body: %s, %w", r.url, r.body, err)
	}

	// 组装header
	for key, value := range r.header {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &nethttp.Client{Transport: &nethttp.Transport{
		TLSClientConfig: r.tLSClientConfig,
	}, Timeout: r.timeout}
	resp, err := client.Do(req)
	if nil != err {
		return fmt.Errorf("request do err, param: %v, %w", req, err)
	}
	defer func() {
		if err = resp.Body.Close(); nil != err {
			fmt.Println(err)
			err = fmt.Errorf("resp body close err, %w", err)
		}
	}()

	// 完整Response
	if r.response != nil {
		r.response.StatusCode = resp.StatusCode
	}

	// 读取请求
	bodyByte, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return fmt.Errorf("request read err, bodyByte: %v, %w", bodyByte, err)
	}

	// 解析请求值
	err = json.Unmarshal(bodyByte, &result)
	if nil != err {
		return fmt.Errorf("request json un err, result: %v, %w", result, err)
	}

	return
}

func (r *http) Get(ctx context.Context, url string, result interface{}, opts ...Option) error {

	// withMethod, WithPreDeal
	opts = append(opts, WithMethod(MethodGet), WithPreDeal(func(r *Parameter) error {
		// 组装url
		params := neturl.Values{}
		netUrl, err := neturl.Parse(url)
		if err != nil {
			return fmt.Errorf("get json ma err, param: %s, %w", url, err)
		}
		for key, value := range r.param {
			// todo 这儿可以优化
			params.Add(key, fmt.Sprintf("%v", value))
		}
		netUrl.RawQuery = params.Encode()
		r.url = netUrl.String()
		if r.log != nil {
			r.log.Println("Get url", r.url)
		}
		return nil
	}))

	return r.request(ctx, url, result, opts...)
}

func (r *http) Post(ctx context.Context, url string, param map[string]interface{}, result interface{}, opts ...Option) error {

	// withMethod, withParam, WithPreDeal
	opts = append(opts, WithMethod(MethodPost), WithParam(param), WithPreDeal(func(r *Parameter) error {
		if nil == param {
			return nil
		}
		// 组装param
		data, err := json.Marshal(param)
		if nil != err {
			return fmt.Errorf("post json ma err, param: %s, %w", param, err)
		}
		r.SetBody(bytes.NewBuffer(data))
		if r.log != nil {
			r.log.Println("Post url", r.url, string(data))
		}
		return nil
	}))

	return r.request(ctx, url, result, opts...)
}

func (r *http) Put(ctx context.Context, url string, param map[string]interface{}, result interface{}, opts ...Option) error {

	// withMethod, withParam, WithPreDeal
	opts = append(opts, WithMethod(MethodPut), WithParam(param), WithPreDeal(func(r *Parameter) error {
		if nil == param {
			return nil
		}
		// 组装param
		data, err := json.Marshal(param)
		if nil != err {
			return fmt.Errorf("put json ma err, param: %s, %w", param, err)
		}
		r.SetBody(bytes.NewBuffer(data))
		if r.log != nil {
			r.log.Println("Put url", r.url, string(data))
		}
		return nil
	}))

	return r.request(ctx, url, result, opts...)
}

func (r *http) Delete(ctx context.Context, url string, result interface{}, opts ...Option) error {

	// withMethod, WithPreDeal
	opts = append(opts, WithMethod(MethodDelete), WithPreDeal(func(r *Parameter) error {
		// 组装url
		params := neturl.Values{}
		netUrl, err := neturl.Parse(url)
		if err != nil {
			return fmt.Errorf("get json ma err, param: %s, %w", url, err)
		}
		for key, value := range r.param {
			// todo 这儿可以优化
			params.Add(key, fmt.Sprintf("%v", value))
		}
		netUrl.RawQuery = params.Encode()
		r.url = netUrl.String()
		if r.log != nil {
			r.log.Println("Delete url", r.url)
		}
		return nil
	}))

	return r.request(ctx, url, result, opts...)
}

func Get(ctx context.Context, url string, result interface{}, opts ...Option) error {
	return NewHttp().Get(ctx, url, result, opts...)
}

func Post(ctx context.Context, url string, param map[string]interface{}, result interface{}, opts ...Option) error {
	return NewHttp().Post(ctx, url, param, result, opts...)
}

func Put(ctx context.Context, url string, param map[string]interface{}, result interface{}, opts ...Option) error {
	return NewHttp().Put(ctx, url, param, result, opts...)
}

func Delete(ctx context.Context, url string, result interface{}, opts ...Option) error {
	return NewHttp().Delete(ctx, url, result, opts...)
}
