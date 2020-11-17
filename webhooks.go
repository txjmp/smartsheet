// Create and manage webhooks
// See smartpgms/server.go for webhook handlers

package smartsheet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type webHookRequest struct {
	Name          string   `json:"name"`
	CallbackUrl   string   `json:"callbackUrl"`
	Scope         string   `json:"scope"`
	ScopeObjectId int64    `json:"scopeObjectId"`
	Events        []string `json:"events"`
	Version       int      `json:"version"`
	SubScope      struct {
		ColumnIds []int64 `json:"columnIds,omitempty"`
	} `json:"subscope,omitempty"`
}

// Create WebHook
func CreateWebHook(sheet *SheetInfo, name string, columnNames ...string) (int64, error) {

	hookReq := webHookRequest{
		Name:          name,
		CallbackUrl:   "https://cheepcode.com/smartsheet/" + name,
		Scope:         "sheet",
		ScopeObjectId: sheet.SheetId,
		Events:        []string{"*.*"},
		Version:       1,
	}
	// optionally, specify columns that trigger webhook call
	if len(columnNames) > 0 {
		hookReq.SubScope.ColumnIds = make([]int64, len(columnNames))
		for i, colName := range columnNames {
			col, found := sheet.ColumnsByName[colName]
			if !found {
				log.Panicln("CreateWebHook bad colName", colName)
			}
			hookReq.SubScope.ColumnIds[i] = col.Id
		}
	}
	reqBytes, _ := json.Marshal(hookReq)
	reqBody := bytes.NewReader(reqBytes)

	fmt.Println(string(reqBytes))

	url := basePath + "/webhooks"
	req, _ := http.NewRequest("POST", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := DoRequest(req)
	if err != nil {
		fmt.Println("xxx CreateWebHook request failed", err)
	}
	defer httpResp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	fmt.Println(string(responseJSON))

	var webHooksResponse struct {
		Message    string `json:"message"`
		ResultCode int    `json:"resultCode"`
		Result     struct {
			Id int64 `json:"id"`
		} `json:"result"`
	}
	err = json.Unmarshal(responseJSON, &webHooksResponse)
	if err != nil {
		log.Panicln("xxx CreateWebHook Unmarshal Response failed", err)
	}
	fmt.Printf("%+v\n\n", webHooksResponse)

	return webHooksResponse.Result.Id, err
}

func EnableWebHook(webHookId int64) error {

	enableReq := map[string]bool{"enabled": true}

	reqBytes, _ := json.Marshal(enableReq)
	reqBody := bytes.NewReader(reqBytes)

	fmt.Println(string(reqBytes))

	url := fmt.Sprintf(basePath+"/webhooks/%d", webHookId)
	fmt.Println("url", url)

	req, _ := http.NewRequest("PUT", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := DoRequest(req)
	if err != nil {
		fmt.Println("xxx CreateWebHook, Enable WebHook failed", err)
	}
	defer httpResp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	fmt.Println("-- Enable Response\n", string(responseJSON))

	return err
}

func GetWebHook(webHookId int64) error {

	url := fmt.Sprintf(basePath+"/webhooks/%d", webHookId)
	fmt.Println("url", url)

	req, _ := http.NewRequest("GET", url, nil)

	httpResp, err := DoRequest(req)
	if err != nil {
		fmt.Println("xxx GetWebHook failed", err)
	}
	defer httpResp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	fmt.Println("-- Response\n", string(responseJSON))

	return err
}

func DeleteWebHook(webHookId int64) error {

	url := fmt.Sprintf(basePath+"/webhooks/%d", webHookId)
	fmt.Println("url", url)

	req, _ := http.NewRequest("DELETE", url, nil)

	httpResp, err := DoRequest(req)
	if err != nil {
		fmt.Println("xxx DeleteWebHook failed", err)
	}
	defer httpResp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	fmt.Println("-- Response\n", string(responseJSON))

	return err
}
