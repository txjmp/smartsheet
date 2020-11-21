package smartsheet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
)

// SheetInfo contains information about a sheet and methods for interacting with it.
// See Load() method for details on what is loaded.
type SheetInfo struct {
	SheetId        int64
	SheetName      string
	WorkspaceId    int64
	WorkspaceName  string
	ColumnsById    map[int64]Column  // sheet columns indexed by Column Id
	ColumnsByName  map[string]Column // sheet columns indexed by Column Title
	ColumnsByIndex map[int]Column    // sheet columns indexed by Column Position, 1st col has index of 0
	Rows           []Row             // rows returned by Load method
	NewRows        []Row             // used by AddRow & UploadNewRows methods
	UpdateRows     []Row             // used by UpdateRow & UploadUpdateRows methods
}

// Load method downloads sheet info by calling GetSheet func and pulling data from the returned sheet info.
// Optional GetSheetOptions is defined in options.go.
// If only specific columns are needed, options.ColumnNames are converted to ColumnIds
func (she *SheetInfo) Load(sheetId int64, options *GetSheetOptions) error {

	// if specified, convert columnNames to columnIds
	if options != nil && len(options.ColumnNames) > 0 {
		options.ColumnIds = make([]int64, len(options.ColumnNames))
		for i, colName := range options.ColumnNames {
			column, found := she.ColumnsByName[colName]
			if !found {
				log.Println("SheetInfo.Load Invalid ColName in options", colName)
				return errors.New("Invalid ColName - " + colName)
			}
			options.ColumnIds[i] = column.Id
		}
	}
	sheet, err := GetSheet(sheetId, options)
	if err != nil {
		log.Println("SheetInfo.load failed", she.SheetName, she.SheetId, err)
		return err
	}
	she.SheetId = sheet.Id
	she.SheetName = sheet.Name
	she.WorkspaceId = sheet.Workspace.Id
	she.WorkspaceName = sheet.Workspace.Name
	she.ColumnsById = make(map[int64]Column)
	she.ColumnsByName = make(map[string]Column)
	she.ColumnsByIndex = make(map[int]Column)
	she.Rows = sheet.Rows

	for _, column := range sheet.Columns {
		she.ColumnsById[column.Id] = column
		she.ColumnsByName[column.Title] = column
		she.ColumnsByIndex[column.Index] = column
	}
	return nil
}

// MatchSheet compares this sheetInfo instance to another instance and returns true if they match.
// Rows are not included in the comparison.
// Useful to determine if a sheet's attributes have changed compared to a previous version.
func (she *SheetInfo) MatchSheet(base *SheetInfo) bool {
	if she.SheetId != base.SheetId {
		log.Println("Sheet Mismatch - SheetId", she.SheetId, base.SheetId)
		return false
	}
	if she.SheetName != base.SheetName {
		log.Println("Sheet Mismatch - SheetName", she.SheetName, base.SheetName)
		return false
	}
	if len(she.ColumnsById) != len(base.ColumnsById) {
		log.Println("Sheet Mismatch - Column Count", len(she.ColumnsById), len(base.ColumnsById))
		return false
	}
	for baseColId, baseColumn := range base.ColumnsById {
		sheetColumn, found := she.ColumnsById[baseColId]
		if !found {
			log.Println("Sheet Mismatch - ColumnId in base, Not in Sheet", baseColumn.Title, baseColId)
			return false
		}
		if sheetColumn.Title != baseColumn.Title {
			log.Printf("Sheet Mismatch - Column Title, Expecting %s, Got %s", baseColumn.Title, sheetColumn.Title)
			return false
		}
		if sheetColumn.Type != baseColumn.Type {
			log.Printf("Sheet Mismatch - Column Type, Expecting %s, Got %s", baseColumn.Type, sheetColumn.Type)
			return false
		}
	}

	// add code to check workspace id, name if in base

	return true
}

// Show displays SheetInfo values in easy to read format.
// To limit number of rows shown, use optional rowLimit.
func (she *SheetInfo) Show(rowLimit ...int) {
	fmt.Println("Sheet Name:", she.SheetName, "Sheet Id:", she.SheetId)
	fmt.Println("Workspace Name:", she.WorkspaceName, "Workspace Id:", she.WorkspaceId)

	fmt.Println("--- COLUMNS ---")
	for index := 0; index < len(she.ColumnsByIndex); index++ {
		column, _ := she.ColumnsByIndex[index]
		fmt.Printf("%2d %15.15s %15.15s %d \n", column.Index, column.Title, column.Type, column.Id)
	}

	fmt.Println("--- ROWS ---")

	rowCount := len(she.Rows)
	fmt.Println("Total Row Count is", rowCount)
	if len(rowLimit) > 0 && rowLimit[0] < rowCount {
		rowCount = rowLimit[0]
	}
	for i := 0; i < rowCount; i++ {
		row := she.Rows[i]
		fmt.Printf("Row %d, id: %d --- \n", i+1, row.Id)
		for _, cell := range row.Cells {
			name := she.ColumnsById[cell.ColumnId].Title
			fmt.Printf("%15s %v \n", name, cell.Value)
		}
	}
}

// AddRow adds a row to SheetInfo.NewRows.
// A row consists of slice of Cell objects and optional locked indicator.
// Do not set lockRow unless new row is to be locked.
// All added rows are processed in a batch using UploadNewRows() method.
func (she *SheetInfo) AddRow(newRow Row) error {
	trace("SheetInfo.AddRow")
	// load Cell.ColumnId using Cell.ColName
	for i := 0; i < len(newRow.Cells); i++ {
		colName := newRow.Cells[i].ColName
		column, found := she.ColumnsByName[colName]
		if !found {
			log.Println("ERROR - SheetInfo.AddRow column not found", she.SheetName, colName)
			return errors.New("Invalid ColumnName - " + colName)
		}
		newRow.Cells[i].ColumnId = column.Id
	}
	if she.NewRows == nil { // set to nil by UploadNewRows
		she.NewRows = make([]Row, 0, 100)
	}
	she.NewRows = append(she.NewRows, newRow)
	return nil
}

// UpdateRow adds a row to SheetInfo.UpdateRows.
// A row consists of rowId, slice of Cell objects, and locked indicator.
// Leave locked field as nil, unless lock status is to be changed.
// All updated rows are processed in a batch using UploadUpdateRows() method.
func (she *SheetInfo) UpdateRow(updtRow Row) error {
	trace("SheetInfo.UpdateRow")
	// load Cell.ColumnId using Cell.colName
	for i := 0; i < len(updtRow.Cells); i++ {
		colName := updtRow.Cells[i].ColName
		column, found := she.ColumnsByName[colName]
		if !found {
			log.Println("ERROR - SheetInfo.UpdateRow column not found", she.SheetName, colName)
			return errors.New("Invalid ColumnName - " + colName)
		}
		updtRow.Cells[i].ColumnId = column.Id
	}
	if she.UpdateRows == nil { // set to nil by UploadUpdateRows
		she.UpdateRows = make([]Row, 0, 100)
	}
	she.UpdateRows = append(she.UpdateRows, updtRow)
	return nil
}

// UploadNewRows adds new rows to sheet using SheetInfo.NewRows.
// After process is complete, NewRows is set to nil.
// If location is nil, rows added to bottom of sheet.
// If rowLevelField is specified, each group of child rows will be indented (using SetParentId), based on value of rowLevelField.
// Currently parent rows should contain "0" and child rows should contain "1" in this field/column.
func (she *SheetInfo) UploadNewRows(location *RowLocation, rowLevelField ...string) (*AddUpdtRowsResponse, error) {
	trace("UploadNewRows")
	if len(she.NewRows) == 0 {
		return nil, nil
	}
	locMap := map[string]interface{}{"toBottom": true}
	if location != nil {
		locMap = CreateLocationMap(location) // see util.go
	}
	// -- Create Request Body ----------------
	type reqItem map[string]interface{}
	reqData := make([]reqItem, 0, len(she.NewRows))

	for _, newRow := range she.NewRows {
		item := make(reqItem)
		item["cells"] = newRow.Cells
		if newRow.Locked != nil { // newRow.Locked is *bool
			item["locked"] = *newRow.Locked // dereference, returns value referenced by pointer
		}
		for k, v := range locMap { // set row location attributes, all rows use same location
			item[k] = v
		}
		reqData = append(reqData, item)
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows", she.SheetId)
	req := Post(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respJSON, _ := ioutil.ReadAll(resp.Body)

	if len(she.NewRows) == 1 { // response.Result is 1 row (not a slice) when adding 1 row
		apiResp1 := new(AddUpdtRowResponse) // same response object when adding or updating row
		err = json.Unmarshal(respJSON, apiResp1)
		if err != nil {
			log.Println("ERROR - UploadAddRows Unmarshal Response for Single Row Failed", err)
			return nil, err
		}
		she.NewRows = nil
		apiResp := AddUpdtRowsResponse{
			Message:    apiResp1.Message,
			ResultCode: apiResp1.ResultCode,
			Result:     []Row{apiResp1.Result},
		}
		return &apiResp, nil
	}

	apiResp := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	err = json.Unmarshal(respJSON, apiResp)
	if err != nil {
		log.Println("ERROR - UploadAddRows Unmarshal Response Failed", err)
		return nil, err
	}

	defer func() {
		she.NewRows = nil
	}()

	if len(rowLevelField) == 0 {
		return apiResp, nil
	}
	// -------------------------------------------------------------------
	// IF OPTIONAL ROWLEVELFIELD SPECIFIED, SET PARENTID ON CHILD ROWS
	//   parent rows: Level 0
	//   child rows: Level 1
	//   child rows must be immediately after parent row in prev api response
	debugLn("Set ParentId on Child Rows ---")
	var parentId int64
	var childIds []int64
	for _, row := range apiResp.Result {
		rowLevel, err := she.GetRowLevel(row, rowLevelField[0])
		if err != nil {
			return apiResp, err
		}
		debugLn("rowLevel", rowLevel)
		if rowLevel == "0" { // if header row
			if len(childIds) > 0 {
				err = SetParentId(she, parentId, childIds) // indent child rows for prev parent
				childIds = make([]int64, 0, 20)
				if err != nil {
					break
				}
			}
			parentId = row.Id
			continue
		}
		if parentId != 0 && rowLevel == "1" {
			childIds = append(childIds, row.Id)
		}
	}
	if len(childIds) > 0 {
		err = SetParentId(she, parentId, childIds) // indent child rows for prev parent
	}
	return apiResp, err
}

// getRowLevel returns the value of cell containing a rows parent-child indicator.
// Parm rowLevelField is the column name, for example "Level".
// If cell does not exist, empty string is returned.
func (she *SheetInfo) GetRowLevel(row Row, rowLevelField string) (string, error) {
	column, found := she.ColumnsByName[rowLevelField]
	if !found {
		log.Println("ERROR - SheetInfo.GetRowLevel invalid rowLevelFld", rowLevelField)
		return "", errors.New("Invalid RowLevel Field")
	}
	rowLevel := ""
	for _, cell := range row.Cells {
		if cell.ColumnId == column.Id {
			rowLevel = cell.Value.(string)
			break
		}
	}
	return rowLevel, nil
}

// UploadUpdateRows updates rows using SheetInfo.UpdateRows.
// After process is complete, UpdateRows is set to nil.
// If location is nil, row position is not changed.
func (she *SheetInfo) UploadUpdateRows(location *RowLocation) (*AddUpdtRowsResponse, error) {
	trace("SheetInfo.UploadUpdateRows")

	var locMap map[string]interface{}
	if location != nil {
		locMap = CreateLocationMap(location) // see util.go
	}
	// -- Create Request Body ----------------
	type reqItem map[string]interface{}
	reqData := make([]reqItem, 0, len(she.UpdateRows))

	for _, updateRow := range she.UpdateRows {
		item := make(reqItem)
		item["id"] = strconv.FormatInt(updateRow.Id, 10) // api expects row id to be a string, don't know why
		if len(updateRow.Cells) > 0 {
			item["cells"] = updateRow.Cells
		}
		if updateRow.Locked != nil { // updateRow.Locked is *bool
			item["locked"] = *updateRow.Locked // dereference, returns value referenced by pointer
		}
		for k, v := range locMap { // set row location attributes, all rows use same location
			item[k] = v
		}
		reqData = append(reqData, item)
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows", she.SheetId)
	req := Put(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respJSON, _ := ioutil.ReadAll(resp.Body)

	apiResp := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	err = json.Unmarshal(respJSON, apiResp)
	if err != nil {
		log.Println("ERROR - UploadUpdateRows Unmarshal Response Failed", err)
		return nil, err
	}
	she.UpdateRows = nil
	return apiResp, nil
}

// CreateCrossSheetReference creates an external-sheet-reference required for cross sheet formulas.
// The CrossSheetReference parameter specifies the sheet, rows, and columns.
func (she *SheetInfo) CreateCrossSheetReference(ref *CrossSheetReference) error {
	trace("CreateCrossSheetReference")

	endPoint := fmt.Sprintf("/sheets/%d/crosssheetreferences", she.SheetId)
	req := Post(endPoint, ref, nil)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := DoRequest(req)
	if err != nil {
		fmt.Println("ERROR - CreateCrossSheetReference request failed", err)
	}
	defer httpResp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	debugLn(string(responseJSON))
	return err
}

// Store saves SheetInfo instance as json encrypted file in indented (readable) format.
func (she *SheetInfo) Store(filePath string) error {
	jsonData, err := json.MarshalIndent(she, "", "  ")
	if err != nil {
		log.Println("ERROR - Store Failed", err)
		return err
	}
	err = ioutil.WriteFile(filePath, jsonData, 0644)
	return err
}

// Restore loads SheetInfo instance from json encrypted file created by Store method.
func (she *SheetInfo) Restore(filePath string) error {
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Println("ERROR - Restore Failed", err)
		return err
	}
	err = json.Unmarshal(jsonData, she)
	return err
}

// ===================================================
/*
// AddIndentRow
func (she *SheetInfo) AddIndentRow(rowId int64) {
	if she.IndentRows == nil {
		she.IndentRows = make([]int64, 0, 100)
	}
	she.IndentRows = append(she.IndentRows, rowId)
}

// UploadIndentRows   API doesn't currently support bulk indent operations
func (she *SheetInfo) UploadIndentRows() (*AddUpdtRowsResponse, error) {
	log.Println("--- SheetInfo.UploadIndentRows ---")

	// -- Create Request Body ----------------
	type reqItem struct {
		Id     int64 `json:"id"`
		Indent int   `json:"indent"`
	}
	request := make([]reqItem, 0, len(she.IndentRows))

	for _, rowId := range she.IndentRows {
		request = append(request, reqItem{Id: rowId, Indent: 1})
	}
	reqBytes, _ := json.Marshal(request)
	fmt.Println("request body ---")
	fmt.Println(string(reqBytes))

	// -- Process Upload Request -----
	url := fmt.Sprintf(basePath+"/sheets/%d/rows", she.SheetId)
	fmt.Println("url", url)

	reqBody := bytes.NewReader(reqBytes)
	req, _ := http.NewRequest("PUT", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := DoRequest(req)
	if err != nil {
		fmt.Println("xxx request failed", err)
	}
	defer httpResp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	response := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	json.Unmarshal(responseJSON, response)

	she.IndentRows = nil
	return response, err
}
*/
