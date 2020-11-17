# Code Info

The purpose of this document is to describe the organization and techniques used to implement functionality.

## URL Parameters
Sometimes the API wants values passed as query parameters in the URL. The Get, Post, Put, Delete funcs in request.go handle this process. They accept a map[string]string parameter containing the key:value combinations. If a key has multiple values, all values must be joined together, separated by commas. FYI, Go's URL.Values type has an Add method that allows multiple values to be added to the same key. Unfortunately the Values.Encode method creates a separate key=value(for each value) in the url query string (which Smartsheet does not like).

## Request Data
For most POST and PUT requests, data is placed into the http request body. The Post, Put funcs in request.go handle this process. The calling func passes the data in whatever format the API requires. For an example, see the Example Code - CopyRows func section below.

## Go Files

* apitypes.go - primary api types: column, cell, row, sheet, etc.
* options.go - types CopyOptions, MoveOptions, GetSheetOptions, RowLocation
* request.go - Get, Post, Put, Delete, DoRequest funcs
* row.go - GetRow, AddRow, UpdateRow, DeleteRows funcs
* sheetinfo.go - SheetInfo type and methods
* smartsheet.go - GetSheet, RowValues, CellInfo, CopyRows, MoveRows, SetParentId, AttachFile,UrlToRow, GetSheetRows funcs
* util.go - CreateLocationMap func
* webhooks.go - CreateWebHook, EnableWebHook, GetWebHook, DeleteWebHook funcs

## Example Code - CopyRows Func
```
// CopyRows copies specified rows from 1 sheet to another.
// CopyOptions indicates what elements are included. If nil, only the row cells are copied.
func CopyRows(fromSheetId int64, rowIds []int64, toSheetId int64, options *CopyOptions) error {
	// -----------------------------------------
    // build request data
    // -----------------------------------------
	var reqData struct {
		RowIds []int64 `json:"rowIds"`
		To     struct {
			SheetId int64 `json:"sheetId"`
		} `json:"to"`
	}
	reqData.RowIds = rowIds
	reqData.To.SheetId = toSheetId
	// -------------------------------------------
    // build url parameter map using CopyOptions
    // -------------------------------------------
	var urlParms map[string]string  // urlParms has value of nil
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
	// -------------------------------------------
    // create url end point
    // -------------------------------------------
	endPoint := fmt.Sprintf("/sheets/%d/rows/copy", fromSheetId)

	// -------------------------------------------
    // call request builder
    // -------------------------------------------
	req := Post(endPoint, reqData, urlParms)
	req.Header.Set("Content-Type", "application/json")

	// -------------------------------------------
    // execute request
    // -------------------------------------------
	resp, err := DoRequest(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
```

## Example Code - Update A Row
Update Status column of a row in the "Tasks" sheet.
Also change it's location to last child of different parent row and lock it.
```
var sheetTasks *SheetInfo  // loaded by other code

location := RowLocation{ ParentId:parentRowId, ToBottom:true }  // parentRowId is int64 var

updateCells := make([]NewCell,1)  // this example is updating 1 cell in the row

updateCells[0] := NewCell{     // create update cell
    ColName: "Status",
    Value:   "Hold",
}
locked := true

UpdateRow(sheetTasks, rowId, updateCells, &location, locked )
```