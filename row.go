// Contains row related funcs not handled by SheetInfo methods.
// GetRow - returns 1 row from sheet
// AddRow - adds 1 row to sheet (use SheetInfo for multiple rows)
// UpdateRow - updates 1 row in sheet (use SheetInfo for multiple rows)
// DeleteRows - deletes 1 or more rows

package smartsheet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

// GetRow returns specified row from sheet.
// ### add code to handle row not found
func GetRow(sheetId, rowId int64) (*Row, error) {
	trace("GetRow")

	urlParms := make(map[string]string)
	urlParms["exclude"] = "nonexistentCells"

	endPoint := fmt.Sprintf("/sheets/%d/rows/%d", sheetId, rowId)
	req := Get(endPoint, urlParms)

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respJSON, _ := ioutil.ReadAll(resp.Body)

	row := new(Row)
	err = json.Unmarshal(respJSON, row)
	if err != nil {
		log.Panicln("GetRow JSON Unmarshal Error - ", err)
	}
	return row, err
}

// AddRow adds 1 row to specified sheet.
// If location is nil, row added to bottom of sheet.
// SheetInfo is used to convert columnNames to columnIds and must contain SheetId.
func AddRow(sheet *SheetInfo, newRow Row, location *RowLocation) (*AddUpdtRowResponse, error) {
	trace("AddRow")

	// load Cell.ColumnId using Cell.colName
	for i := 0; i < len(newRow.Cells); i++ {
		colName := newRow.Cells[i].ColName
		column, found := sheet.ColumnsByName[colName]
		if !found {
			log.Println("ERROR - SheetInfo.AddRow column not found", sheet.SheetName, colName)
			return nil, errors.New("Invalid ColumnName - " + colName)
		}
		newRow.Cells[i].ColumnId = column.Id
	}

	// create row location map
	locMap := map[string]interface{}{"toBottom": true}
	if location != nil {
		locMap = CreateLocationMap(location) // see util.go
	}

	// -- create request body ----------------
	reqData := make(map[string]interface{})
	reqData["cells"] = newRow.Cells
	if newRow.Locked != nil { // newRow.Locked is *bool
		reqData["locked"] = *newRow.Locked // dereference, returns value referenced by pointer
	}
	for k, v := range locMap { // set row location attributes, all rows use same location
		reqData[k] = v
	}

	// -- create & process api request -----------
	endPoint := fmt.Sprintf("/sheets/%d/rows", sheet.SheetId)
	req := Post(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respJSON, _ := ioutil.ReadAll(resp.Body)
	debugLn(string(respJSON))

	apiResp := new(AddUpdtRowResponse) // add 1 row resp.Result is type Row not []Row
	err = json.Unmarshal(respJSON, apiResp)
	if err != nil {
		log.Println("ERROR - AddRow Unmarshal Response Failed", err)
		return nil, err
	}
	return apiResp, nil
}

// UpdataRow updates 1 row in specified sheet.
// If location is nil, row location is not changed.
// SheetInfo is used to convert columnNames to columnIds and must contain SheetId.
// Omit lockRow parm to leave lock status unchanged.
func UpdateRow(sheet *SheetInfo, updtRow Row, location *RowLocation) (*AddUpdtRowsResponse, error) {
	trace("UpdateRow")

	// -- load Cell.ColumnId using Cell.colName -------------
	for i := 0; i < len(updtRow.Cells); i++ {
		colName := updtRow.Cells[i].ColName
		column, found := sheet.ColumnsByName[colName]
		if !found {
			log.Println("ERROR - SheetInfo.AddRow column not found", sheet.SheetName, colName)
			return nil, errors.New("Invalid ColumnName - " + colName)
		}
		updtRow.Cells[i].ColumnId = column.Id
	}

	// -- create row location map ----------------
	var locMap map[string]interface{}
	if location != nil {
		locMap = CreateLocationMap(location) // see util.go
	}

	// -- create request body ----------------
	reqData := make(map[string]interface{})
	reqData["id"] = strconv.FormatInt(updtRow.Id, 10) // api expects row id to be a string, don't know why
	reqData["cells"] = updtRow.Cells
	if updtRow.Locked != nil { // newRow.Locked is *bool
		reqData["locked"] = *updtRow.Locked // dereference, returns value referenced by pointer
	}
	for k, v := range locMap { // set row location attributes, all rows use same location
		reqData[k] = v
	}

	// -- create api request & process ------------------
	endPoint := fmt.Sprintf("/sheets/%d/rows", sheet.SheetId)
	req := Put(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respJSON, _ := ioutil.ReadAll(resp.Body)
	debugLn(string(respJSON))
	apiResp := new(AddUpdtRowsResponse) // update response.Result is always type []Row
	err = json.Unmarshal(respJSON, apiResp)
	if err != nil {
		log.Println("ERROR - UpdateRow Unmarshal Response Failed", err)
		return nil, err
	}
	return apiResp, nil
}

// DeleteRows removes specified rowsIds from sheet.
func DeleteRows(sheetId int64, rowIds ...int64) error {

	ids := make([]string, len(rowIds))
	for i, id := range rowIds {
		ids[i] = strconv.FormatInt(id, 10)
	}
	urlParms := make(map[string]string)
	urlParms["ids"] = strings.Join(ids, ",")

	endPoint := fmt.Sprintf("/sheets/%d/rows", sheetId)
	req := Delete(endPoint, urlParms)

	resp, err := DoRequest(req)
	if err != nil {
		log.Println("ERROR - DeleteRows Failed", err)
		return err
	}
	defer resp.Body.Close()
	respJSON, _ := ioutil.ReadAll(resp.Body)
	debugLn("DeleteRows ---")
	debugLn(string(respJSON))
	return nil
}
