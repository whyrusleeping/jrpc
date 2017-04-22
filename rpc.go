package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

var DefaultUser string
var DefaultPass string

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

type Response struct {
	Result     interface{} `json:"result"`
	ResultType interface{} `json:"-"`
	Error      *Error      `json:"error,omitempty"`
}

func (r *Response) UnmarshalJSON(b []byte) error {
	i := struct {
		Result *json.RawMessage
		Error  *Error
	}{}

	err := json.Unmarshal(b, &i)
	if err != nil {
		return err
	}

	r.Error = i.Error

	if r.ResultType == nil && i.Result != nil {
		return json.Unmarshal(*i.Result, &r.Result)
	}

	val := reflect.New(reflect.TypeOf(r.ResultType)).Interface()
	err = json.Unmarshal(*i.Result, val)
	if err != nil {
		return err
	}

	r.Result = val
	return nil
}

type Request struct {
	JsonRPC string      `json:"jsonrpc,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      int         `json:"id"`
}

type Client struct {
	Host string
	User string
	Pass string
}

var DefaultClient = &Client{
	Host: "http://localhost:8232",
}

func Do(obj *Request, out interface{}) error {
	return DefaultClient.Do(obj, out)
}

func (c *Client) Do(obj *Request, out interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	body := bytes.NewReader(data)

	req, err := http.NewRequest("POST", c.Host, body)
	if err != nil {
		return err
	}
	req.Header["Content-Type"] = []string{"application/json"}

	// auth auth baby
	if c.User != "" {
		req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.User+":"+c.Pass)))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("failed to connect to daemon, is it running?")
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("%s: %s", resp.Status, string(data))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
