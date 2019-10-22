package main

import (
	"fmt"
	"io"

	"github.com/valyala/fasthttp"
)

var fhClient *fasthttp.Client

type fasthttpReq struct{}

func NewFastHTTPRequester() *fasthttpReq {
	fhClient = &fasthttp.Client{
		MaxConnsPerHost: nRoutines,
		ReadTimeout:     timeout,
		WriteTimeout:    timeout,
	}
	return &fasthttpReq{}
}

func (f *fasthttpReq) Exists(id string) (bool, error) {
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.Header.SetMethod("HEAD")
	req.SetRequestURI(fmt.Sprintf("https://imgur.com/%s", id))

	err := fhClient.Do(req, res)
	if err != nil {
		return false, err
	}

	switch res.StatusCode() {
	case fasthttp.StatusOK:
		return true, nil
	case fasthttp.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("http status %d", res.Header.StatusCode())
	}
}

func (f *fasthttpReq) StreamTo(id string, w io.Writer) error {
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.SetRequestURI(fmt.Sprintf("https://i.imgur.com/%s.jpg", id))

	err := fhClient.Do(req, res)
	if err != nil {
		return err
	}

	switch res.StatusCode() {
	case fasthttp.StatusOK:
		return res.BodyWriteTo(w)
	default:
		return fmt.Errorf("http status %d", res.StatusCode())
	}
}
