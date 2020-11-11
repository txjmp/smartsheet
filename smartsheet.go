package smartsheet

import (
	"bytes"
	"encoding/json"
	"errors"
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

var RequestDelay time.Duration = 1 * time.Second // delay between API requests, maximum of 100 requests per minute

type m bson.M

var tokenIndex int
var tokens = []string{
	"Bearer iaio1on056ri3ajvqt6wcjxjwq",
}

const basePath = "https://api.smartsheet.com/2.0"

const (
	EGNYTE = "EGNYTE"
	LINK   = "LINK"
)

// RowLocation indicates where a row should be added or moved to.
type RowLocation struct {
	ParentId, SiblingId           int64 // 0 indicates no parent or sibling, only one can be used
	ToTop, ToBottom, AboveSibling bool  // only one should be true, ToBottom is default when adding rows to sheet without parent
	Indent, Outdent               int   // to activate, load either with value of 1
}

// GetSheetOptions determines what rows and columns are returned by GetSheet func.
// If no attributes set, all rows and columns returned.
type GetSheetOptions struct {
	RowIds            []int64   // include only specific rows, added to url query parameters
	RowsModifiedSince time.Time // include only rows modified since specific time
	RowsModifiedMins  int       // include only rows where modified-time within x minutes before current time
	ColumnNames       []string  // used by sheetInfo.Load to get columnIds, not used by GetSheet func
	ColumnIds         []int64   // include only specified columns
}

var NoRows = &GetSheetOptions{RowIds: []int64{0}} // for convenience, to specify no rows should be returned

// GetSheet downloads specified sheet info based on GetSheetOptions.
// Typically Load() method of SheetInfo instance is used to call GetSheet.
// If options is nil, all rows and columns are requested.
// Cells never containing a value are automatically excluded.
func GetSheet(sheetId int64, options *GetSheetOptions) (*Sheet, error) {
	trace("GetSheet")
	if options == nil {
		options = new(GetSheetOptions)
	}
	debugObj(options)

	var err error
	url := fmt.Sprintf(basePath+"/sheets/%d", sheetId)
	req, _ := http.NewRequest("GET", url, nil)

	qryParms := req.URL.Query() // Get a copy of the url query string
	qryParms.Add("exclude", "nonexistentCells")

	if len(options.RowIds) > 0 {
		rowIds := make([]string, len(options.RowIds))
		for i, rowId := range options.RowIds {
			rowIds[i] = fmt.Sprintf("%d", rowId)
		}
		qryParms.Add("rowIds", strings.Join(rowIds, ","))
	}

	if len(options.ColumnIds) > 0 {
		colIds := make([]string, len(options.ColumnIds))
		for i, colId := range options.ColumnIds {
			colIds[i] = fmt.Sprintf("%d", colId)
		}
		qryParms.Add("columnIds", strings.Join(colIds, ","))
	}

	if !options.RowsModifiedSince.IsZero() {
		qryParms.Add("rowsModifiedSince", options.RowsModifiedSince.Format(time.RFC3339))
	}

	if options.RowsModifiedMins > 0 {
		d := time.Duration(options.RowsModifiedMins) * time.Minute // convert mins to duration type & compute duration
		rowsModifiedSince := time.Now().Add(-d).Format(time.RFC3339)
		debugLn("rowsModifiedSince", rowsModifiedSince)
		qryParms.Add("rowsModifiedSince", rowsModifiedSince)
	}

	req.URL.RawQuery = qryParms.Encode() // Encode and assign back to the original query.
	debugLn("URL: ", req.URL)

	resp, err := performRequest(req, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseJSON, _ := ioutil.ReadAll(resp.Body)
	debugLn(string(responseJSON))

	response := new(Sheet)

	err = json.Unmarshal(responseJSON, response)
	if err != nil {
		log.Panicln("Smartsheet LoadSheet JSON Unmarshal Error - ", err)
	}
	debugObj(response)

	return response, err
}

// GetSheetRows creates csv file containing row data, 1st line is column headers.
func GetSheetRows(sheetId int64, filePath string) error {

	url := fmt.Sprintf(basePath+"/sheets/%d", sheetId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/csv")

	resp, err := performRequest(req, false)
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

// GetRow returns specified row from sheet.
// ### add code to handle row not found
func GetRow(sheetId, rowId int64) (*Row, error) {
	trace("GetRow")

	url := fmt.Sprintf(basePath+"/sheets/%d/rows/%d", sheetId, rowId)
	req, _ := http.NewRequest("GET", url, nil)

	qryParms := req.URL.Query() // Get a copy of the url query string
	qryParms.Add("exclude", "nonexistentCells")
	req.URL.RawQuery = qryParms.Encode() // Encode and assign back to the original query.

	debugLn("URL: ", req.URL.RawQuery)

	resp, err := performRequest(req, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseJSON, _ := ioutil.ReadAll(resp.Body)
	response := new(Row)

	err = json.Unmarshal(responseJSON, response)
	if err != nil {
		log.Panicln("Smartsheet GetRow JSON Unmarshal Error - ", err)
	}
	debugObj(response)

	return response, err
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

// CopyRows copies specified rows from 1 sheet to another.
// Optional CopyOptions indicates what elements, attached to each row, are included.
// If CopyOptions is nil, only the row cells are copied.
func CopyRows(fromSheetId int64, rowIds []int64, toSheetId int64, options *CopyOptions) (string, error) {
	trace("CopyRows")
	var request struct {
		RowIds []int64 `json:"rowIds"`
		To     struct {
			SheetId int64 `json:"sheetId"`
		} `json:"to"`
	}
	request.RowIds = rowIds
	request.To.SheetId = toSheetId

	reqBytes, _ := json.Marshal(request)
	reqBody := bytes.NewReader(reqBytes)
	debugLn("request body ---")
	debugLn(string(reqBytes))

	url := fmt.Sprintf(basePath+"/sheets/%d/rows/copy", fromSheetId)
	debugLn("url", url)

	req, _ := http.NewRequest("POST", url, reqBody)

	if options != nil {
		qryParms := req.URL.Query() // Get a copy of the query string.
		if options.All {
			qryParms.Add("include", "all")
		} else {
			if options.Attachments {
				qryParms.Add("include", "attachments")
			}
			if options.Children {
				qryParms.Add("include", "children")
			}
			if options.Discussions {
				qryParms.Add("include", "discussions")
			}
		}
		req.URL.RawQuery = qryParms.Encode() // Encode and assign back to the original query.
		debugLn("CopyRows url qrystring", req.URL.RawQuery)
	}
	httpResp, err := performRequest(req, false)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()
	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	return string(responseJSON), err // currently just returning the response as a string, for debugging
}

// MoveOptions is used by MoveRows to indicate what elements (in addition to cells) are copied to the destination sheet.
type MoveOptions struct {
	Attachments, Discussions bool // Child rows are always moved
}

// MoveRows moves specified rows from 1 sheet to another.
// Optional MoveOptions indicates what elements, attached to each row, are included. Child rows are always included.
// If MoveOptions is nil, only the row cells are moved.
func MoveRows(fromSheetId int64, rowIds []int64, toSheetId int64, options *MoveOptions) (string, error) {
	trace("MoveRows")
	var request struct {
		RowIds []int64 `json:"rowIds"`
		To     struct {
			SheetId int64 `json:"sheetId"`
		} `json:"to"`
	}
	request.RowIds = rowIds
	request.To.SheetId = toSheetId

	reqBytes, _ := json.Marshal(request)
	reqBody := bytes.NewReader(reqBytes)
	debugLn("request body ---")
	debugLn(string(reqBytes))

	url := fmt.Sprintf(basePath+"/sheets/%d/rows/move", fromSheetId)
	debugLn("url", url)

	req, _ := http.NewRequest("POST", url, reqBody)

	if options != nil {
		qryParms := req.URL.Query() // Get a copy of the query string.
		if options.Attachments {
			qryParms.Add("include", "attachments")
		}
		if options.Discussions {
			qryParms.Add("include", "discussions")
		}
		req.URL.RawQuery = qryParms.Encode() // Encode and assign back to the original query.
		debugLn("MoveRows url qrystring", req.URL.RawQuery)
	}
	httpResp, err := performRequest(req, false)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()
	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	return string(responseJSON), err // currently just returning the response as a string, for debugging
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
	// Create Request Body
	type reqItem struct {
		Id       int64 `json:"id"`
		ParentId int64 `json:"parentId"`
		ToBottom *bool `json:"toBottom,omitempty"`
	}
	request := make([]reqItem, len(childIds))

	for i, childId := range childIds {
		request[i] = reqItem{Id: childId, ParentId: parentId}
	}
	if len(toBottom) > 0 && len(childIds) == 1 && toBottom[0] {
		request[0].ToBottom = &isTrue
	}
	reqBytes, _ := json.Marshal(request)
	debugLn("request body ---")
	debugLn(string(reqBytes))

	// Process Upload Request
	url := fmt.Sprintf(basePath+"/sheets/%d/rows", sheet.SheetId)
	debugLn("url", url)

	reqBody := bytes.NewReader(reqBytes)
	req, _ := http.NewRequest("PUT", url, reqBody)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := performRequest(req, false)
	defer httpResp.Body.Close()

	if err != nil {
		responseJSON, _ := ioutil.ReadAll(httpResp.Body)
		log.Println("SetParentId Failed ---\n", string(responseJSON))
	}
	return err
}

// GetCrossSheetRefs displays cross sheet ref info for sheet.
func GetCrossSheetRefs(sheetId int64) {
	url := fmt.Sprintf(basePath+"/sheets/%d/crosssheetreferences", sheetId)
	req, _ := http.NewRequest("GET", url, nil)
	httpResp, err := performRequest(req, false)
	if err != nil {
		fmt.Println("xxx GetCrossSheetRefs failed", err)
	}
	defer httpResp.Body.Close()
	responseJSON, _ := ioutil.ReadAll(httpResp.Body)
	fmt.Println("-- CrossSheetRefs --\n", string(responseJSON))
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
	debugLn("url", url)

	req, _ := http.NewRequest("POST", url, file)
	req.Header.Set("Content-Type", "") // let Smartsheet figure out from fileName
	req.Header.Set("Content-Disposition", "attachment; filename="+fileName)
	req.Header.Set("Content-Length", fileSize)

	httpResp, err := performRequest(req, false)
	defer httpResp.Body.Close()

	debugObj(httpResp)

	return err
}

// AttachUrlToRow attaches url link to a row.
func AttachUrlToRow(sheetId, rowId int64, fileName, attachmentType, linkUrl string) error {
	trace("AttachUrlToRow")

	//'{"name":"Search Engine", "description": "A popular search engine", "attachmentType":"LINK", "url":"http://www.google.com"}'

	var request = struct {
		Name           string `json:"name"`
		AttachmentType string `json:"attachmentType"` // LINK, BOX_COM, DROPBOX, EGNYTE, EVERNOTE, GOOGLE_DRIVE, ONEDRIVE
		Url            string `json:"url"`
	}{
		Name:           fileName,
		AttachmentType: attachmentType,
		Url:            linkUrl,
	}
	reqBytes, _ := json.Marshal(&request)
	reqBody := bytes.NewReader(reqBytes)

	url := fmt.Sprintf(basePath+"/sheets/%d/rows/%d/attachments", sheetId, rowId)
	req, _ := http.NewRequest("POST", url, reqBody)
	req.Header.Set("Content-Type", "application/json") // let Smartsheet figure out from fileName

	httpResp, err := performRequest(req, false)
	defer httpResp.Body.Close()

	debugObj(httpResp)

	return err
}

func performRequest(req *http.Request, closeBody bool) (*http.Response, error) {
	tokenIndex += 1
	if tokenIndex >= len(tokens) {
		tokenIndex = 0
	}
	req.Header.Set("Authorization", tokens[tokenIndex])
	client := http.Client{}
	client.Timeout = time.Second * 120
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Smartsheet Error, HTTP Request Failed - ", err)
		log.Println(resp.Header)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp, errors.New(resp.Status)
	}
	if closeBody {
		resp.Body.Close()
	}
	time.Sleep(RequestDelay) // limit number of requests per minute
	return resp, nil
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
		_, err = performRequest(req, true)

		return err

	}}
*/
