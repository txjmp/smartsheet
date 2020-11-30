package smartsheet

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const basePath = "https://api.smartsheet.com/2.0"

var Token string

var RequestDelay time.Duration = 1 * time.Second // delay between API requests, maximum of 100 requests per minute

// Get returns a GET http.Request object.
// UrlParms are added to the URL as Query parameters.
func Get(endPoint string, urlParms map[string]string) *http.Request {
	url := basePath + endPoint
	req, _ := http.NewRequest("GET", url, nil)
	if len(urlParms) > 0 {
		qryParms := req.URL.Query()
		for key, val := range urlParms {
			qryParms.Add(key, val)
		}
		req.URL.RawQuery = qryParms.Encode()
	}
	debugLn("GET - ", req.URL.RequestURI())
	return req
}

// Post returns a POST http.Request object.
// UrlParms are added to the URL as Query parameters.
func Post(endPoint string, data interface{}, urlParms map[string]string) *http.Request {

	reqBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Panicln("Post - Cannot Marshal Request Data", err)
	}
	debugLn("POST request data ---")
	debugLn(string(reqBytes))

	reqBody := bytes.NewReader(reqBytes)

	url := basePath + endPoint
	req, _ := http.NewRequest("POST", url, reqBody)
	if len(urlParms) > 0 {
		qryParms := req.URL.Query()
		for key, val := range urlParms {
			qryParms.Add(key, val)
		}
		req.URL.RawQuery = qryParms.Encode()
	}
	debugLn("POST - ", req.URL.RequestURI())
	return req
}

// Put returns a PUT http.Request object.
// UrlParms are added to the URL as Query parameters.
func Put(endPoint string, data interface{}, urlParms map[string]string) *http.Request {

	reqBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Panicln("Put - Cannot Marshal Request Data", err)
	}
	debugLn("PUT request data ---")
	debugLn(string(reqBytes))

	reqBody := bytes.NewReader(reqBytes)

	url := basePath + endPoint
	req, _ := http.NewRequest("PUT", url, reqBody)
	if len(urlParms) > 0 {
		qryParms := req.URL.Query()
		for key, val := range urlParms {
			qryParms.Add(key, val)
		}
		req.URL.RawQuery = qryParms.Encode()
	}
	debugLn("PUT - ", req.URL.RequestURI())
	return req
}

// Delete returns a DELETE http.Request object.
// UrlParms are added to the URL as Query parameters.
func Delete(endPoint string, urlParms map[string]string) *http.Request {
	url := basePath + endPoint
	req, _ := http.NewRequest("DELETE", url, nil)
	if len(urlParms) > 0 {
		qryParms := req.URL.Query()
		for key, val := range urlParms {
			qryParms.Add(key, val)
		}
		req.URL.RawQuery = qryParms.Encode()
	}
	debugLn("DELETE - ", req.URL.RequestURI())
	return req
}

// DoRequest executes the supplied http request and returns the http response.
// If an error occurs, response info is logged.
// After request completes, execution is paused (based on RequestDelay value) to throttle request frequency.
func DoRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", Token)
	client := http.Client{}
	client.Timeout = time.Second * 120
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Println("Smartsheet Error, HTTP Request Failed - ", err)
		log.Println("Http Response StatusCode", resp.StatusCode)
		log.Println("-- resp Header -----")
		log.Println(resp.Header)
		if resp.Body != nil {
			respBody, _ := ioutil.ReadAll(resp.Body)
			log.Println("-- resp Body -----")
			log.Println(string(respBody))
			resp.Body.Close()
		}
		return nil, errors.New("Smartsheet Http API Request Failed - See Log For Details")
	}
	time.Sleep(RequestDelay) // limit number of requests per minute
	return resp, nil
}
