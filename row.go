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
func AddRow(sheet *SheetInfo, newCells []NewCell, location *RowLocation, lockRow ...bool) (*AddUpdtRowsResponse, error) {
	trace("AddRow")

	rowCells := make([]Cell, len(newCells))
	for i, newCell := range newCells {
		column, found := sheet.ColumnsByName[newCell.ColName]
		if !found {
			log.Println("ERROR - AddRow column not found", sheet.SheetName, newCell.ColName)
			return nil, errors.New("Invalid ColumnName - " + newCell.ColName)
		}
		rowCells[i] = Cell{
			ColumnId:  column.Id,
			Formula:   newCell.Formula,   // only Formula or Value can be loaded, "" will be omitted
			Value:     newCell.Value,     // only Formula or Value can be loaded, nil will be omitted
			Hyperlink: newCell.Hyperlink, // nil will be omitted
		}
	}
	locMap := map[string]interface{}{"toBottom": true}
	if location != nil {
		locMap = CreateLocationMap(location) // see util.go
	}
	// -- Create Request Body ----------------
	reqData := make(map[string]interface{})
	reqData["cells"] = rowCells
	if len(lockRow) > 0 && lockRow[0] { // false not used for new rows
		reqData["locked"] = true
	}
	for k, v := range locMap { // set row location attributes, all rows use same location
		reqData[k] = v
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows", sheet.SheetId)
	req := Post(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respJSON, _ := ioutil.ReadAll(resp.Body)
	result := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	err = json.Unmarshal(respJSON, result)
	if err != nil {
		log.Println("ERROR - AddRow Unmarshal Response Failed", err)
		return nil, err
	}
	return result, nil
}

// UpdataRow updates 1 row in specified sheet.
// If location is nil, row location is not changed.
// SheetInfo is used to convert columnNames to columnIds and must contain SheetId.
// Omit lockRow parm to leave lock status unchanged.
func UpdateRow(sheet *SheetInfo, rowId int64, newCells []NewCell, location *RowLocation, lockRow ...bool) (*AddUpdtRowsResponse, error) {
	trace("UpdateRow")

	rowCells := make([]Cell, len(newCells))
	for i, newCell := range newCells {
		column, found := sheet.ColumnsByName[newCell.ColName]
		if !found {
			log.Println("ERROR - UpdateRow column not found", sheet.SheetName, newCell.ColName)
			return nil, errors.New("Invalid ColumnName - " + newCell.ColName)
		}
		rowCells[i] = Cell{
			ColumnId:  column.Id,
			Formula:   newCell.Formula,   // only Formula or Value can be loaded, "" will be omitted
			Value:     newCell.Value,     // only Formula or Value can be loaded, nil will be omitted
			Hyperlink: newCell.Hyperlink, // nil will be omitted
		}
	}
	var locMap map[string]interface{}
	if location != nil {
		locMap = CreateLocationMap(location) // see util.go
	}
	// -- Create Request Body ----------------
	reqData := make(map[string]interface{})
	reqData["id"] = strconv.FormatInt(rowId, 10) // api expects row id to be a string, don't know why
	reqData["cells"] = rowCells
	if len(lockRow) > 0 
		reqData["locked"] = lockRow[0]
	}
	for k, v := range locMap { // set row location attributes, all rows use same location
		reqData[k] = v
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows", sheet.SheetId)
	req := Put(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respJSON, _ := ioutil.ReadAll(resp.Body)
	result := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	err = json.Unmarshal(respJSON, result)
	if err != nil {
		log.Println("ERROR - UpdateRow Unmarshal Response Failed", err)
		return nil, err
	}
	return result, nil
}

// DeleteRows removes specified rowsIds from sheet.
func DeleteRows(sheetId int64, rowIds ...int64) error {

	ids := make([]string,len(rowIds))
	for i, id := range rowIds {
		ids[i] = strconv.FormatInt(id, 10)
	}
	urlParms := make(map[string]string)
	urlParms["ids"] = strings.Join(ids,",")

	endPoint := fmt.Sprintf("/sheets/%d/rows", sheet.SheetId)
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