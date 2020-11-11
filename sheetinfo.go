package smartsheet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

// NewCell is used when adding or updating rows.
// See SheetInfo AddRow() and UpdateRow() methods.
type NewCell struct {
	ColName   string
	Formula   string      // only formula or value can be loaded
	Value     interface{} // if hyperlink, value is what's displayed in cell
	Hyperlink *Hyperlink
}

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
// Optional GetSheetOptions is defined in smartsheet.go.
// If only specific columns are needed, options.ColumnNames are converted to ColumnIds
func (she *SheetInfo) Load(sheetId int64, options *GetSheetOptions) {

	if options == nil {
		options = new(GetSheetOptions)
	}
	options.ColumnIds = make([]int64, len(options.ColumnNames))
	for i, colName := range options.ColumnNames {
		column := she.ColumnsByName[colName]
		options.ColumnIds[i] = column.Id
	}
	sheet, err := GetSheet(sheetId, options)
	if err != nil {
		log.Panicln("SheetInfo.load failed", she.SheetName, she.SheetId, err)
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
// A row consists of slice of NewCell objects and optional lockRow indicator.
// Do not include lockRow unless new row is to be locked.
// All added rows are processed in a batch using UploadNewRows() method.
func (she *SheetInfo) AddRow(newCells []NewCell, lockRow ...bool) error {

	var locked *bool                    // if lockRow not specified, locked is nil (not false)
	if len(lockRow) > 0 && lockRow[0] { // false not used for new rows
		locked = &isTrue
	}
	newRow := Row{
		Cells:  make([]Cell, len(newCells)),
		Locked: locked, // if nil, will be omitted from api call
	}
	for i, newCell := range newCells {
		column, found := she.ColumnsByName[newCell.ColName]
		if !found {
			log.Println("ERROR - SheetInfo.AddRow column not found", she.SheetName, newCell.ColName)
			return errors.New("Invalid ColumnName - " + newCell.ColName)
		}
		newRow.Cells[i] = Cell{
			ColumnId:  column.Id,
			Formula:   newCell.Formula,   // only Formula or Value can be loaded, "" will be omitted
			Value:     newCell.Value,     // only Formula or Value can be loaded, nil will be omitted
			Hyperlink: newCell.Hyperlink, // nil will be omitted
		}
	}
	if she.NewRows == nil { // set to nil by UploadNewRows
		she.NewRows = make([]Row, 0, 100)
	}
	she.NewRows = append(she.NewRows, newRow)
	return nil
}

// UpdateRow adds a row to SheetInfo.UpdateRows.
// A row consists of rowId, slice of NewCell objects, and optional lockRow indicator.
// Only include lockRow if row is to be locked(true) or unlocked(false).
// All updated rows are processed in a batch using UploadUpdateRows() method.
func (she *SheetInfo) UpdateRow(rowId int64, newCells []NewCell, lockRow ...bool) {

	var locked *bool // if lockRow not specified, locked is nil (not false)
	if len(lockRow) > 0 {
		if lockRow[0] {
			locked = &isTrue
		} else {
			locked = &isFalse
		}
	}
	newRow := Row{
		Id:     rowId,
		Cells:  make([]Cell, len(newCells)),
		Locked: locked, // if nil, will be omitted from api call
	}
	for i, newCell := range newCells {
		column, found := she.ColumnsByName[newCell.ColName]
		if !found {
			log.Panicln("ERROR - SheetInfo.UpdateRow column not found", she.SheetName, newCell.ColName)
		}
		newRow.Cells[i] = Cell{
			ColumnId:  column.Id,
			Formula:   newCell.Formula,   // only Formula or Value can be loaded, "" will be omitted
			Value:     newCell.Value,     // only Formula or Value can be loaded, nil will be omitted
			Hyperlink: newCell.Hyperlink, // nil will be omitted
		}
	}
	if she.UpdateRows == nil { // set to nil by UploadUpdateRows
		she.UpdateRows = make([]Row, 0, 100)
	}
	she.UpdateRows = append(she.UpdateRows, newRow)
}

// UploadNewRows adds new rows to sheet using SheetInfo.NewRows.
// After process is complete, NewRows is set to nil.
// If location is nil, rows added to bottom of sheet.
// If rowLevelField is specified, each group of child rows will be indented (using SetParentId), based on value of rowLevelField.
// Currently parent rows should contain "0" and child rows should contain "1" in this field/column.
func (she *SheetInfo) UploadNewRows(location *RowLocation, rowLevelField ...string) (*AddUpdtRowsResponse, error) {
	trace("UploadNewRows")

	if location == nil {
		location = &RowLocation{ToBottom: true}
	}
	// -- Create Request Body ----------------
	type reqItem map[string]interface{}
	request := make([]reqItem, 0, len(she.NewRows))

	for _, newRow := range she.NewRows {
		item := make(reqItem)
		item["cells"] = newRow.Cells
		if newRow.Locked != nil { // newRow.Locked is *bool
			item["locked"] = *newRow.Locked // dereference, returns value referenced by pointer
		}
		if location.ToTop {
			item["toTop"] = true
		}
		if location.ToBottom {
			item["toBottom"] = true
		}
		if location.ParentId != 0 {
			item["parentId"] = location.ParentId
		}
		if location.SiblingId != 0 {
			item["siblingId"] = location.SiblingId
		}
		if location.AboveSibling {
			item["above"] = true
		}
		if location.Indent != 0 {
			item["indent"] = 1
		}
		if location.Outdent != 0 {
			item["outdent"] = 1
		}
		request = append(request, item)
	}

	reqBytes, _ := json.Marshal(request)
	debugLn("request body ---")
	debugLn(string(reqBytes))

	// -- Process Upload Request -----
	url := fmt.Sprintf(basePath+"/sheets/%d/rows", she.SheetId)
	debugLn("url", url)

	reqBody := bytes.NewReader(reqBytes)
	req, _ := http.NewRequest("POST", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := performRequest(req, false)
	defer httpResp.Body.Close()
	if err != nil {
		log.Printf("%+v\n", httpResp)
		//log.Panicln("ERROR - SheetInfo.UploadNewRows failed", err)
	}
	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	response := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	json.Unmarshal(responseJSON, response)

	debugLn("response ---")
	debugObj(response)

	// -------------------------------------------------------------------
	// IF OPTIONAL ROWLEVELFIELD SPECIFIED, SET PARENTID ON CHILD ROWS
	//   parent rows: Level 0
	//   child rows: Level 1
	//   child rows must be immediately after parent row in response
	if len(rowLevelField) > 0 {
		debugLn("Set ParentId on Child Rows ---")
		var parentId int64
		var childIds []int64
		for _, row := range response.Result {
			rowLevel := she.GetRowLevel(row, rowLevelField[0])
			debugLn("rowLevel", rowLevel)
			if rowLevel == "0" { // if header row
				if len(childIds) > 0 {
					SetParentId(she, parentId, childIds) // indent child rows for prev parent
				}
				parentId = row.Id
				childIds = make([]int64, 0, 20)
				continue
			}
			if parentId != 0 && rowLevel == "1" {
				childIds = append(childIds, row.Id)
			}
		}
		if len(childIds) > 0 {
			SetParentId(she, parentId, childIds) // indent child rows for prev parent
		}
	}
	she.NewRows = nil
	return response, err
}

// getRowLevel returns the value of cell containing a rows parent-child indicator.
// Parm rowLevelField is the column name, for example "Level".
// Currently method UploadNewRows() expects Parent row value to be "0" and Child row value to be "1".
func (she *SheetInfo) GetRowLevel(row Row, rowLevelField string) string {
	column, found := she.ColumnsByName[rowLevelField]
	if !found {
		log.Panicln("SheetInfo.GetRowLevel invalid rowLevelFld", rowLevelField)
	}
	for _, cell := range row.Cells {
		if cell.ColumnId == column.Id {
			return cell.Value.(string)
		}
	}
	return ""
}

// UploadUpdateRows updates rows using SheetInfo.UpdateRows.
// After process is complete, UpdateRows is set to nil.
// If location is nil, row position is not changed.
func (she *SheetInfo) UploadUpdateRows(location *RowLocation) (*AddUpdtRowsResponse, error) {
	trace("SheetInfo.UploadUpdateRows")

	// -- Create Request Body ----------------
	type reqItem map[string]interface{}
	request := make([]reqItem, 0, len(she.UpdateRows))

	for _, updateRow := range she.UpdateRows {
		item := make(reqItem)
		item["id"] = strconv.FormatInt(updateRow.Id, 10) // api expects row id to be a string, don't know why
		if len(updateRow.Cells) > 0 {
			item["cells"] = updateRow.Cells
		}
		if updateRow.Locked != nil { // updateRow.Locked is *bool
			item["locked"] = *updateRow.Locked // dereference, returns value referenced by pointer
		}
		if location.ToTop {
			item["toTop"] = true
		}
		if location.ToBottom {
			item["toBottom"] = true
		}
		if location.ParentId != 0 {
			item["parentId"] = location.ParentId
		}
		if location.SiblingId != 0 {
			item["siblingId"] = location.SiblingId
		}
		if location.AboveSibling {
			item["above"] = true
		}
		if location.Indent != 0 {
			item["indent"] = 1
		}
		if location.Outdent != 0 {
			item["outdent"] = 1
		}
		request = append(request, item)
	}
	reqBytes, _ := json.Marshal(request)
	debugLn("request body ---")
	debugLn(string(reqBytes))

	// -- Process Upload Request -----
	url := fmt.Sprintf(basePath+"/sheets/%d/rows", she.SheetId)
	fmt.Println("url", url)

	reqBody := bytes.NewReader(reqBytes)
	req, _ := http.NewRequest("PUT", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := performRequest(req, false)
	defer httpResp.Body.Close()
	if err != nil {
		log.Printf("%+v\n", httpResp)
		//log.Panicln("ERROR - SheetInfo.UploadUpdateRows failed", err)
	}
	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	response := new(AddUpdtRowsResponse) // same response object when adding or updating rows
	json.Unmarshal(responseJSON, response)

	she.UpdateRows = nil
	return response, err
}

// CopyOptions is used by CopyRows to indicate what elements (in addition to cells) are copied to the destination sheet.
type CopyOptions struct {
	All, Attachments, Children, Discussions bool // specify All or any mix of other options
}

// CreateCrossSheetReference creates an external-sheet-reference required for cross sheet formulas.
// The CrossSheetReference parameter specifies the sheet, rows, and columns.
func (she *SheetInfo) CreateCrossSheetReference(ref *CrossSheetReference) error {
	trace("CreateCrossSheetReference")

	reqBytes, _ := json.Marshal(ref)
	reqBody := bytes.NewReader(reqBytes)
	debugLn("request body ---")
	debugLn(string(reqBytes))

	url := fmt.Sprintf(basePath+"/sheets/%d/crosssheetreferences", she.SheetId)
	req, _ := http.NewRequest("POST", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := performRequest(req, false)
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

	httpResp, err := performRequest(req, false)
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
