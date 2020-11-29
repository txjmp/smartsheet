package smartsheet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
)

type EmailRecipient map[string]interface{} // key: "email" or "groupId", val: address or groupId number

// EmailRowsObj is the object sent by EmailRows func via API.
type EmailRowsObj struct {
	SendTo             []EmailRecipient `json:"sendTo"`
	Subject            string           `json:"subject"`
	Message            string           `json:"message"`
	CCMe               bool             `json:"ccMe"`
	RowIds             []int64          `json:"rowIds"`
	ColumnIds          []int64          `json:"columnIds,omitempty"`
	IncludeAttachments bool             `json:"includeAttachments"`
	IncludeDiscussions bool             `json:"includeDiscussions"`
}

// EmailRows emails sheet rows using values in EmailRowsObj parm.
func EmailRows(sheetId int64, reqData EmailRowsObj) error {

	endPoint := fmt.Sprintf("/sheets/%d/rows/emails", sheetId)
	req := Post(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Message    string `json:"message"`
		ResultCode int    `json:"resultCode"`
	}
	respJSON, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(respJSON, &apiResp)
	if err != nil {
		log.Println("ERROR EmailRows Unmarshal Response Failed", err)
		log.Println(string(respJSON))
		return err
	}
	if apiResp.ResultCode != 0 {
		log.Println("ERROR EmailRows Was Not Successful")
		log.Println("Message:", apiResp.Message, "Code:", apiResp.ResultCode)
		return errors.New("EmailRows Failed, See Log For Details")
	}
	return nil
}
