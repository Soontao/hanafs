package hana

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/imroc/req"
)

const keyCSRFTokenHeader = "x-csrf-token"

const keyAuthorization = "Authorization"

const keyContentType = "Content-Type"

const valueRequired = "required"

// Client type
type Client struct {
	uri           *url.URL
	token         string
	req           *req.Req
	sslVerify     bool
	baseDirectory string
	tokenLock     sync.RWMutex
}

// get csrf token value
func (c *Client) getToken() string {
	c.tokenLock.RLock()
	defer c.tokenLock.RUnlock()
	return c.token
}

// set csrf token value
func (c *Client) setToken(tokenValue string) {
	c.tokenLock.Lock()
	defer c.tokenLock.Unlock()
	c.token = tokenValue
}

func (c *Client) formatURI(path string) string {
	port := c.uri.Port()
	// default to 443 port
	if len(port) == 0 {
		port = "443"
	}
	return fmt.Sprintf("https://%s:%s%s", c.uri.Host, port, path)
}

func isCSRFTokenError(response *http.Response) bool {
	return (response.StatusCode == http.StatusForbidden &&
		strings.ToLower(response.Header.Get(keyCSRFTokenHeader)) == valueRequired)
}

// Rename file or move file
func (c *Client) Rename(old, new string, dir bool) (err error) {
	// only support rename
	// if users want to move from directory to another, maybe a delete & create walkaround required.

	oldPath, oldName := filepath.Split(old)
	newPath, newName := filepath.Split(new)

	if oldPath != newPath {
		return ErrOpNotAllowed
	}

	if oldName == newName {
		return
	}

	payload := map[string]interface{}{
		// old file location
		"Location": c.formatDtFilePath(old),
		"Target":   newName,
	}

	header := req.Header{
		keyContentType:     "application/json;charset=UTF-8",
		"X-Create-Options": "move,no-overwrite",
	}

	res, err := c.request("POST", c.formatDtFilePath(oldPath), req.BodyJSON(&payload), header)

	if err != nil {
		return err
	}

	if res.Response().StatusCode != 201 {
		return ErrOpNotAllowed
	}

	return
}

func (c *Client) request(method, path string, infos ...interface{}) (*req.Resp, error) {

	// format url
	url := c.formatURI(path)

	password, _ := c.uri.User.Password()

	header := req.Header{
		keyCSRFTokenHeader: c.getToken(),
		keyAuthorization:   basicAuth(c.uri.User.Username(), password),
	}

	infos = append(infos, header)

	// do request
	resp, err := c.req.Do(method, url, infos...)

	if resp != nil {

		if isCSRFTokenError(resp.Response()) {
			// try refresh csrf token
			if err := c.fetchCSRFToken(); err != nil {
				return nil, err
			}
			// update token
			header[keyCSRFTokenHeader] = c.getToken()
			// re process request
			resp, err = c.req.Do(method, url, infos...)
		}

		response := resp.Response()

		switch {
		case response.StatusCode > 400:
			err = errors.New(response.Status)
		}

	}

	if err != nil {
		return nil, err
	}

	return resp, err

}

func (c *Client) fetchCSRFToken() error {

	password, _ := c.uri.User.Password()
	header := req.Header{
		keyCSRFTokenHeader: "fetch",
		keyAuthorization:   basicAuth(c.uri.User.Username(), password),
	}

	resp, err := c.req.Head(c.formatURI("/sap/hana/xs/dt/base/file"), header)

	if err != nil {
		return err
	}

	httpResponse := resp.Response()

	status := httpResponse.StatusCode

	token := httpResponse.Header.Get("x-csrf-token")

	if len(token) != 0 && token != "unsafe" {
		c.setToken(token)
	} else {
		switch {
		case 300 <= status && status < 400:
			return errors.New("redirect, please check your credential")
		case 400 <= status && status < 500:
			return errors.New("request is not accepted")
		case 500 <= status && status < 600:
			return errors.New("server is down")
		default:
			return errors.New("could not fetch csrf token, please check your credential")
		}
	}

	return nil
}

func (c *Client) checkCredential() error {
	if err := c.fetchCSRFToken(); err != nil {
		return err
	}
	return nil
}

func (c *Client) checkURIValidate(uri *url.URL) error {
	_, err := net.LookupHost(uri.Host)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) formatDtFilePath(path string) string {
	realPath := strings.ReplaceAll(path, "\\", "/")
	return fmt.Sprintf("/sap/hana/xs/dt/base/file%s%s", c.baseDirectory, realPath)
}

// ReadFile content
func (c *Client) ReadFile(filePath string) ([]byte, error) {
	res, err := c.request(
		"GET",
		c.formatDtFilePath(filePath),
	)

	if err != nil {
		return nil, err
	}

	if res.Response().StatusCode == 404 {
		return nil, ErrFileNotFound
	}

	return res.ToBytes()
}

// Create file or directory
func (c *Client) Create(base, name string, dir bool) error {

	payload := map[string]interface{}{
		"Name":      name,
		"Directory": dir,
	}

	res, err := c.request(
		"POST",
		c.formatDtFilePath(base),
		req.BodyJSON(&payload),
		req.Header{keyContentType: "application/json"},
	)

	if err == nil && res.Response().StatusCode >= 300 {
		err = errors.New(res.Response().Status)
	}

	if err != nil {
		return err
	}

	return nil
}

// WriteFileContent to hana
func (c *Client) WriteFileContent(path string, content []byte) (err error) {

	res, err := c.request(
		"PUT",
		c.formatDtFilePath(path),
		content,
	)

	if err == nil && res.Response().StatusCode >= 300 {
		err = errors.New(res.Response().Status)
	}

	if err != nil {
		return err
	}

	return nil

}

// ReadDirectory information
func (c *Client) ReadDirectory(filePath string) (*DirectoryDetail, error) {

	res, err := c.request(
		"GET",
		c.formatDtFilePath(filePath),
		req.QueryParam{"depth": 1},
	)

	if err != nil {
		return nil, err
	}

	if res.Response().StatusCode == 404 {
		return nil, ErrFileNotFound
	}

	rt := &DirectoryDetail{}

	if err := res.ToJSON(rt); err != nil {
		return nil, err
	}

	return rt, nil

}

func (c *Client) Delete(path string) (rt error) {

	res, err := c.request(
		"DELETE",
		c.formatDtFilePath(path),
	)

	if res.Response().StatusCode == 404 {
		return ErrFileNotFound
	}

	if err != nil {
		return err
	}

	return
}

// Stat func
func (c *Client) Stat(filePath string) (*PathStat, error) {

	rt := &PathStat{}

	query := req.QueryParam{
		"depth": 0,
		"parts": "meta",
	}

	res, err := c.request(
		"GET",
		c.formatDtFilePath(filePath),
		query,
	)

	if err != nil {
		return nil, err
	}

	if res.Response().StatusCode == 404 {
		return nil, ErrFileNotFound
	}

	body, err := res.ToString()

	if gjson.Get(body, "Directory").Bool() {

		dir := &DirectoryMeta{}
		if err := json.Unmarshal([]byte(body), dir); err != nil {
			return nil, err
		}

		rt.Directory = dir.Directory
		rt.Executable = dir.Attributes.Executable
		rt.Archive = dir.Attributes.Archive
		rt.Hidden = dir.Attributes.Hidden
		rt.ReadOnly = dir.Attributes.ReadOnly
		rt.SymbolicLink = dir.Attributes.SymbolicLink

	} else {
		f := &File{}

		if err := json.Unmarshal([]byte(body), f); err != nil {
			return nil, err
		}

		rt.Directory = f.Directory
		rt.Executable = f.Attributes.Executable
		rt.Archive = f.Attributes.Archive
		rt.Hidden = f.Attributes.Hidden
		rt.ReadOnly = f.Attributes.ReadOnly
		rt.SymbolicLink = f.Attributes.SymbolicLink
		rt.Activated = f.Attributes.SapBackPack.Activated

		rt.TimeStamp = f.SapBackPack.ActivatedAt

	}

	return rt, nil
}

// NewClient for hana
func NewClient(uri *url.URL) (*Client, error) {
	rt := &Client{uri: uri, req: req.New(), baseDirectory: uri.Path, sslVerify: true}

	if err := rt.checkURIValidate(uri); err != nil {
		return nil, err
	}

	if err := rt.checkCredential(); err != nil {
		return nil, err
	}

	rt.req.EnableCookie(true)

	trans, _ := rt.req.Client().Transport.(*http.Transport)

	trans.MaxIdleConns = 50
	trans.TLSHandshakeTimeout = 20 * time.Second
	trans.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return rt, nil
}
