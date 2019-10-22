package main

import (
	"fmt"
	"io"
	"net/http"
)

type vanillaReq struct{}

func NewVanillaRequester() *vanillaReq {
	return &vanillaReq{}
}

func (v *vanillaReq) Exists(id string) (bool, error) {
	res, err := http.Head(fmt.Sprintf("https://imgur.com/%s", id))
	if err != nil {
		return false, err
	}

	switch res.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("http status %s", res.Status)
	}
}

func (v *vanillaReq) StreamTo(id string, w io.Writer) error {
	res, err := http.Get(fmt.Sprintf("https://i.imgur.com/%s.jpg", id))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		_, err := io.Copy(w, res.Body)
		return err
	default:
		return fmt.Errorf("http status %s", res.Status)
	}
}
