package hana

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/imroc/req"
)

const keyCSRFTokenHeader = "x-csrf-token"

const valueRequired = "required"

// Client type
type Client struct {
	uri           *url.URL
	token         string
	req           *req.Req
	sslVerify     bool
	baseDirectory string
}

func (c *Client) formatURI(path string) string {
	port := c.uri.Port()
	if len(port) == 0 {
		port = "443"
	}
	return fmt.Sprintf("https://%s:%s%s", c.uri.Host, port, path)
}

func isCSRFTokenError(response *http.Response) bool {
	return (response.StatusCode == http.StatusForbidden &&
		strings.ToLower(response.Header.Get(keyCSRFTokenHeader)) == valueRequired)
}

func (c *Client) request(method, path string, infos ...interface{}) (*req.Resp, error) {

	// format url
	url := c.formatURI(path)

	// do request
	resp, err := req.Do(method, url, infos...)

	if isCSRFTokenError(resp.Response()) {
		// try refresh csrf token
		if err := c.fetchCSRFToken(); err != nil {
			return nil, err
		}
		// re process request
		resp, err = req.Do(method, url, infos...)
	}

	if err != nil {
		return nil, err
	}

	return resp, err

}

func (c *Client) fetchCSRFToken() error {

	header := req.Header{
		keyCSRFTokenHeader: "fetch",
	}

	resp, err := c.req.Head(c.formatURI("/sap/hana/xs/dt/base/file"), header)

	if err != nil {
		return err
	}

	httpResponse := resp.Response()

	status := httpResponse.StatusCode

	token := httpResponse.Header.Get("x-csrf-token")

	if len(token) != 0 && token != "unsafe" {
		c.token = token
	} else {
		switch {
		case 300 <= status && status < 400:
			return errors.New("redirect, please check your credential")
		case 400 <= status && status < 500:
			return errors.New("request is not accepted")
		case 500 <= status && status < 600:
			return errors.New("server is down")
		default:
			return errors.New("not found csrf token, please check your credential")
		}
	}

	return nil
}

func (c *Client) checkCredential() error {
	return nil
}

func (c *Client) checkURIValidate(uri url.URL) error {

	return nil
}

// NewClient for hana
func NewClient(uri *url.URL) (*Client, error) {
	rt := &Client{uri: uri, req: req.New(), baseDirectory: uri.Path, sslVerify: true}

	if err := rt.checkCredential(); err != nil {
		return nil, err
	}

	return rt, nil
}
