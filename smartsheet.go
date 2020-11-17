package smartsheet

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

var DebugOn bool = false // calling package can turn these on/off as needed
var TraceOn bool = false

var isTrue bool = true // address of this var is used when setting boolean pointer values
var isFalse bool = false

type m bson.M

const (
	EGNYTE = "EGNYTE"
	LINK   = "LINK"
)

// GetSheet downloads specified sheet info based on GetSheetOptions.
// Typically Load() method of SheetInfo instance is used to call GetSheet.
// If options is nil, all rows and columns are requested.
// Cells never containing a value are automatically excluded.
func GetSheet(sheetId int64, options *GetSheetOptions) (*Sheet, error) {
	trace("GetSheet")
	if options == nil {
		options = new(GetSheetOptions)
	}
	debugLn("GetSheetOptions ---")
	debugObj(options)

	endPoint := fmt.Sprintf("/sheets/%d", sheetId)

	urlParms := make(map[string]string)
	urlParms["exclude"] = "nonexistentCells"
	if len(options.RowIds) > 0 {
		rowIds := make([]string, len(options.RowIds))
		for i, rowId := range options.RowIds {
			rowIds[i] = fmt.Sprintf("%d", rowId)
		}
		urlParms["rowIds"] = strings.Join(rowIds, ",")
	}
	if len(options.ColumnIds) > 0 {
		colIds := make([]string, len(options.ColumnIds))
		for i, colId := range options.ColumnIds {
			colIds[i] = fmt.Sprintf("%d", colId)
		}
		urlParms["columnIds"] = strings.Join(colIds, ",")
	}
	if !options.RowsModifiedSince.IsZero() {
		urlParms["rowsModifiedSince"] = options.RowsModifiedSince.Format(time.RFC3339)
	}
	if options.RowsModifiedMins > 0 {
		d := time.Duration(options.RowsModifiedMins) * time.Minute // convert mins to duration type & compute duration
		rowsModifiedSince := time.Now().Add(-d).Format(time.RFC3339)
		debugLn("rowsModifiedSince: ", rowsModifiedSince)
		urlParms["rowsModifiedSince"] = rowsModifiedSince
	}
	req := Get(endPoint, urlParms)
	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respJSON, _ := ioutil.ReadAll(resp.Body)

	sheet := new(Sheet)
	err = json.Unmarshal(respJSON, sheet)
	if err != nil {
		log.Panicln("GetSheet JSON Unmarshal Error - ", err)
	}
	return sheet, err
}

// GetSheetRows creates csv file containing row data, 1st line is column headers.
func GetSheetRows(sheetId int64, filePath string) error {

	endPoint := fmt.Sprintf("/sheets/%d", sheetId)
	req := Get(endPoint, nil)
	req.Header.Set("Accept", "text/csv")

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("Smartsheet GetSheetRows Error, Creating Local File - ", err)
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Println("Smartsheet GetSheetRows Error, Writing Local File - ", err)
	}
	return err
}

// RowValues returns a single row's cell values as map[string]string.
// The key of each entry is column name.
// If cell contains hyperlink, the url is returned as entry value.
// If cell contains multiple values, all values are concatenated into 1 string, ex: "light, sour".
// If cell contains number value, it is converted to string (formatting such as $, commas are not included).
// Cells with no value have entry value of empty string, "".
// Formula cells return the computed value (not the formula).
// Use func CellInfo() to access all cell attributes.
func RowValues(sheet *SheetInfo, row Row) map[string]string {
	trace("RowValues")
	rowValues := make(map[string]string)
	for _, cell := range row.Cells {
		column := sheet.ColumnsById[cell.ColumnId]
		colName := column.Title
		switch {
		case cell.Hyperlink != nil && cell.Hyperlink.Url != "":
			rowValues[colName] = cell.Hyperlink.Url
		case cell.Value == nil:
			rowValues[colName] = ""
		default:
			rowValues[colName] = fmt.Sprintf("%v", cell.Value)
		}
	}
	// load missing columns with "" (cells never having value are not returned by GetSheet() func)
	for colName, _ := range sheet.ColumnsByName {
		if _, found := rowValues[colName]; !found {
			rowValues[colName] = ""
		}
	}
	debugObj(rowValues)
	return rowValues
}

// CellInfo returns pointer to copy of a specific cell in a row.
// Parm columnName determines which cell in row to return. Must be in sheet.ColumnNames.
// Parm row is the row containing the cell. It is not required to be in sheet.Rows.
// Type Cell provides access to all cell attributes, such as formula which is not returned by RowValues().
// Cell is not required to exist for requested columnName.
func CellInfo(sheet *SheetInfo, row Row, columnName string) *Cell {
	response := new(Cell)
	column, found := sheet.ColumnsByName[columnName]
	if !found {
		log.Println("ERROR - CellInfo, columnName not found in sheet.ColumnsByName: ", columnName)
		return nil
	}
	for _, cell := range row.Cells { // range returns copy of value
		if cell.ColumnId == column.Id {
			response = &cell
			break
		}
	}
	return response
}

// CopyRows copies specified rows from 1 sheet to bottom of another (RowLocation not supported).
// Optional CopyOptions indicates what elements, attached to each row, are included.
// If CopyOptions is nil, only the row cells are copied.
func CopyRows(fromSheetId int64, rowIds []int64, toSheetId int64, options *CopyOptions) error {
	trace("CopyRows")
	var reqData struct {
		RowIds []int64 `json:"rowIds"`
		To     struct {
			SheetId int64 `json:"sheetId"`
		} `json:"to"`
	}
	reqData.RowIds = rowIds
	reqData.To.SheetId = toSheetId

	var urlParms map[string]string
	if options != nil {
		ops := make([]string, 0, 3)
		if options.All {
			ops = append(ops, "all")
		} else {
			if options.Attachments {
				ops = append(ops, "attachments")
			}
			if options.Children {
				ops = append(ops, "children")
			}
			if options.Discussions {
				ops = append(ops, "discussions")
			}
		}
		if len(ops) > 0 {
			urlParms = make(map[string]string)
			urlParms["include"] = strings.Join(ops, ",")
		}
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows/copy", fromSheetId)
	req := Post(endPoint, reqData, urlParms)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// MoveRows moves specified rows from 1 sheet to another.
// Optional MoveOptions indicates what elements, attached to each row, are included. Child rows are always included.
// If MoveOptions is nil, only the row cells are moved.
func MoveRows(fromSheetId int64, rowIds []int64, toSheetId int64, options *MoveOptions) error {
	trace("MoveRows")
	var reqData struct {
		RowIds []int64 `json:"rowIds"`
		To     struct {
			SheetId int64 `json:"sheetId"`
		} `json:"to"`
	}
	reqData.RowIds = rowIds
	reqData.To.SheetId = toSheetId

	var urlParms map[string]string
	if options != nil {
		ops := make([]string, 0, 2)
		if options.Attachments {
			ops = append(ops, "attachments")
		}
		if options.Discussions {
			ops = append(ops, "discussions")
		}
		if len(ops) > 0 {
			urlParms = make(map[string]string)
			urlParms["include"] = strings.Join(ops, ",")
		}
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows/move", fromSheetId)
	req := Post(endPoint, reqData, urlParms)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// SetParentId sets parent (indents) specified child rows.
// Parm parentId is rowId of parent row.
// If multiple childIds, row ordering not changed.
// If single childId, optional toBottom can be used. Default location is 1st child of parent.
func SetParentId(sheet *SheetInfo, parentId int64, childIds []int64, toBottom ...bool) error {
	trace("SetParentId")

	if len(childIds) == 0 {
		log.Println("SetParentId - No ChildIds Specified")
		return nil
	}
	type reqItem struct {
		Id       int64 `json:"id"`
		ParentId int64 `json:"parentId"`
		ToBottom *bool `json:"toBottom,omitempty"`
	}
	reqData := make([]reqItem, len(childIds))

	for i, childId := range childIds {
		reqData[i] = reqItem{Id: childId, ParentId: parentId}
	}
	if len(toBottom) > 0 && len(childIds) == 1 && toBottom[0] {
		reqData[0].ToBottom = &isTrue
	}
	endPoint := fmt.Sprintf("/sheets/%d/rows", sheet.SheetId)
	req := Put(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// GetCrossSheetRefs displays cross sheet ref info for sheet.
func GetCrossSheetRefs(sheetId int64) error {
	endPoint := fmt.Sprintf("/sheets/%d/crosssheetreferences", sheetId)
	req := Get(endPoint, nil)

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respJSON, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("-- CrossSheetRefs --\n", string(respJSON))
	return nil
}

// AttachFileToRow attaches file to row.
// Parm filePath specifies local system file to be attached
// File is uploaded to Smartsheet.
// Expensive operation. Counts as 10 interactions. See API Limits documentation for details.
func AttachFileToRow(sheetId, rowId int64, filePath string) error {
	trace("AttachFileToRow")

	fileName := filepath.Base(filePath)
	debugLn("fileName", fileName)

	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Attach File Error, Cannot Open File - ", err)
		return err
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	fileSize := fmt.Sprintf("%d", fileInfo.Size())
	debugLn("fileSize", fileSize)

	url := fmt.Sprintf(basePath+"/sheets/%d/rows/%d/attachments", sheetId, rowId)
	req, _ := http.NewRequest("POST", url, file)
	req.Header.Set("Content-Type", "") // let Smartsheet figure out from fileName
	req.Header.Set("Content-Disposition", "attachment; filename="+fileName)
	req.Header.Set("Content-Length", fileSize)

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// AttachUrlToRow attaches url link to a row.
func AttachUrlToRow(sheetId, rowId int64, fileName, attachmentType, linkUrl string) error {
	trace("AttachUrlToRow")

	//'{"name":"Search Engine", "description": "A popular search engine", "attachmentType":"LINK", "url":"http://www.google.com"}'

	var reqData struct {
		Name           string `json:"name"`
		AttachmentType string `json:"attachmentType"` // LINK, BOX_COM, DROPBOX, EGNYTE, EVERNOTE, GOOGLE_DRIVE, ONEDRIVE
		Url            string `json:"url"`
	}
	reqData.Name = fileName
	reqData.AttachmentType = attachmentType
	reqData.Url = linkUrl

	endPoint := fmt.Sprintf("/sheets/%d/rows/%d/attachments", sheetId, rowId)
	req := Post(endPoint, reqData, nil)
	req.Header.Set("Content-Type", "application/json") // let Smartsheet figure out from fileName

	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func trace(stepName string) {
	if TraceOn {
		fmt.Println("-- " + stepName + " -------------------")
	}
}

func debugLn(args ...interface{}) {
	if DebugOn {
		fmt.Println(args...)
	}
}

func debugObj(obj interface{}) {
	if DebugOn {
		fmt.Printf("%+v\n", obj)
	}
}

/*
func MultiAttachFile() {

		file, err := os.Open(fromPath)
		if err != nil {
			log.Println("Egnyte Uploadfile Error, Cannot Open Local File - ", err)
			return err
		}
		defer file.Close()

		// --- create http request body containing file ---------------
		var body bytes.Buffer
		multiPartWriter := multipart.NewWriter(&body)
		formFilePart, _ := multiPartWriter.CreateFormFile("file", fromPath)
		if _, err = io.Copy(formFilePart, file); err != nil {
			log.Println("Egnyte UploadFile Error, MultiPartWriter Failed - ", err)
			return err
		}
		contentType := multiPartWriter.FormDataContentType()
		multiPartWriter.Close() // cannot use defer for close

		reqBody := bytes.NewReader(body.Bytes())



		req, err := http.NewRequest("POST", basePath+"/fs-content"+toPath, reqBody)

		req.Header.Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", "attachment; filename=Wiki.png")
		_, err = DoRequest(req, true)

		return err

	}}
*/
