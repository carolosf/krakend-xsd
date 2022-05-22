package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
	"io"
	"log"
	"net/http"
	"net/url"
)

func main() {}

func init() {
	log.Println(string(ModifierRegisterer), "loaded plugin")
}

// ModifierRegisterer is the symbol the plugin loader will be looking for. It must
// implement the plugin.Registerer interface
// https://github.com/luraproject/lura/blob/master/proxy/plugin/modifier.go#L71
var ModifierRegisterer = registerer("krakend-xsd")

type registerer string

// RegisterModifiers is the function the plugin loader will call to register the
// modifier(s) contained in the plugin using the function passed as argument.
// f will register the factoryFunc under the name and mark it as a request
// and/or response modifier.
func (r registerer) RegisterModifiers(f func(
	name string,
	factoryFunc func(map[string]interface{}) func(interface{}) (interface{}, error),
	appliesToRequest bool,
	appliesToResponse bool,
)) {
	f(string(r)+"-request", r.requestDump, true, false)
	f(string(r)+"-response", r.responseDump, false, true)
	log.Println(string(r), "registered plugin")
}

var unkownTypeErr = errors.New("unknow request type")

func (r registerer) requestDump(
	cfg map[string]interface{},
) func(interface{}) (interface{}, error) {
	// return the modifier
	log.Println("request dumper injected!!!")

	return func(input interface{}) (interface{}, error) {
		req, ok := input.(RequestWrapper)
		if !ok {
			return nil, unkownTypeErr
		}

		xsdvalidate.Init()
		defer xsdvalidate.Cleanup()
		relCfg := cfg["krakend-xsd"].(map[string]interface{})
		xsdFile, ok := relCfg["xsd_file"].(string)
		if !ok {
			log.Println("Must enter an xsd file path")
		}

		xsdHandler, err := xsdvalidate.NewXsdHandlerUrl(xsdFile, xsdvalidate.ParsErrVerbose)
		if err != nil {
			log.Println("Could not load xsd file: ", xsdFile)
		}
		defer xsdHandler.Free()

		log.Println("intercepting request")
		log.Println("url:", req.URL())
		log.Println("method:", req.Method())
		log.Println("headers:", req.Headers())
		log.Println("params:", req.Params())
		log.Println("query:", req.Query())
		log.Println("path:", req.Path())
		//log.Println("body:", req.Body())

		log.Println("Validate xsd body")

		if xsdHandler == nil {
			return nil, err
		}

		toByte := ConvertToByte(req.Body())
		log.Println("body:", toByte)
		err = xsdHandler.ValidateMem(toByte, xsdvalidate.ParsErrVerbose)
		if err != nil {
			return nil, err
		}

		//convertRequestForModification(req)
		return input, nil
	}
}

func convertRequestForModification(req RequestWrapper) requestWrapper {
	return requestWrapper{
		req.Method(),
		req.URL(),
		req.Query(),
		req.Path(),
		req.Body(),
		req.Params(),
		req.Headers(),
	}
}

func convertToHttpRequest(req RequestWrapper) (*http.Request, error) {
	request, err := http.NewRequest(req.Method(), "http://localhost:8080", req.Body())
	if err != nil {
		return nil, err
	}
	for k, y := range req.Headers() {
		for _, v := range y {
			request.Header.Add(k, v)
		}
	}
	return request, nil
}

func encodeMetadataAsBytes(req RequestWrapper) (bytes.Buffer, error) {
	var metadata bytes.Buffer
	enc := gob.NewEncoder(&metadata)
	err := enc.Encode(requestMetadataWrapper{
		req.Method(),
		req.URL(),
		req.Query(),
		req.Path(),
		req.Params(),
		req.Headers(),
	})
	return metadata, err
}

func ConvertToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func (r registerer) responseDump(
	cfg map[string]interface{},
) func(interface{}) (interface{}, error) {
	// return the modifier
	log.Println("response dumper injected!!!")

	return func(input interface{}) (interface{}, error) {
		resp, ok := input.(ResponseWrapper)
		if !ok {
			return nil, unkownTypeErr
		}

		xsdvalidate.Init()
		defer xsdvalidate.Cleanup()
		relCfg := cfg["krakend-xsd"].(map[string]interface{})
		xsdFile, ok := relCfg["xsd_file"].(string)
		if !ok {
			log.Println("Must enter an xsd file path")
		}

		xsdHandler, err := xsdvalidate.NewXsdHandlerUrl(xsdFile, xsdvalidate.ParsErrVerbose)
		if err != nil {
			log.Println("Could not load xsd file: ", xsdFile)
		}
		defer xsdHandler.Free()

		log.Println("response", cfg)

		log.Println("intercepting response")
		log.Println("is complete:", resp.IsComplete())
		log.Println("headers:", resp.Headers())
		log.Println("status code:", resp.StatusCode())
		//log.Println("data:", resp.Data())

		log.Println("Scan body")

		if xsdHandler == nil {
			return nil, err
		}

		err = xsdHandler.ValidateMem(ConvertToByte(resp.Io()), xsdvalidate.ValidErrDefault)
		if err != nil {
			return nil, err
		}

		//tmp := convertResponseForModification(resp)
		return input, nil
	}
}

func encodeResponseMetadataAsBytes(resp ResponseWrapper) (bytes.Buffer, error) {
	var metadata bytes.Buffer
	enc := gob.NewEncoder(&metadata)
	err := enc.Encode(ResponseMetadataWrapper{
		resp.Headers(),
		resp.StatusCode(),
	})
	return metadata, err
}

func convertResponseForModification(resp ResponseWrapper) responseWrapper {
	return responseWrapper{
		data:       resp.Data(),
		isComplete: resp.IsComplete(),
		metadata: metadataWrapper{
			headers:    resp.Headers(),
			statusCode: resp.StatusCode(),
		},
		io: resp.Io(),
	}
}

// RequestWrapper is an interface for passing proxy request between the lura pipe and the loaded plugins
type RequestWrapper interface {
	Params() map[string]string
	Headers() map[string][]string
	Body() io.ReadCloser
	Method() string
	URL() *url.URL
	Query() url.Values
	Path() string
}

// ResponseWrapper is an interface for passing proxy response between the lura pipe and the loaded plugins
type ResponseWrapper interface {
	Data() map[string]interface{}
	Io() io.Reader
	IsComplete() bool
	Headers() map[string][]string
	StatusCode() int
}

type requestMetadataWrapper struct {
	Method  string
	Url     *url.URL
	Query   url.Values
	Path    string
	Params  map[string]string
	Headers map[string][]string
}

type requestWrapper struct {
	method  string
	url     *url.URL
	query   url.Values
	path    string
	body    io.ReadCloser
	params  map[string]string
	headers map[string][]string
}

func (r *requestWrapper) Method() string               { return r.method }
func (r *requestWrapper) URL() *url.URL                { return r.url }
func (r *requestWrapper) Query() url.Values            { return r.query }
func (r *requestWrapper) Path() string                 { return r.path }
func (r *requestWrapper) Body() io.ReadCloser          { return r.body }
func (r *requestWrapper) Params() map[string]string    { return r.params }
func (r *requestWrapper) Headers() map[string][]string { return r.headers }

func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

type metadataWrapper struct {
	headers    map[string][]string
	statusCode int
}

type ResponseMetadataWrapper struct {
	Headers    map[string][]string
	StatusCode int
}

func (m metadataWrapper) Headers() map[string][]string { return m.headers }
func (m metadataWrapper) StatusCode() int              { return m.statusCode }

type responseWrapper struct {
	data       map[string]interface{}
	isComplete bool
	metadata   metadataWrapper
	io         io.Reader
}

func (r responseWrapper) Data() map[string]interface{} { return r.data }
func (r responseWrapper) IsComplete() bool             { return r.isComplete }
func (r responseWrapper) Io() io.Reader                { return r.io }
func (r responseWrapper) Headers() map[string][]string { return r.metadata.headers }
func (r responseWrapper) StatusCode() int              { return r.metadata.statusCode }