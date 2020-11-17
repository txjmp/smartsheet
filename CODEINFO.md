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

## SheetInfo Type
Contains sheet attributes like id, name, and columns (id,title,type). Columns can accessed by id, name(title), or index(position). Depending on what options were used when loaded, it may also contain, all / some / none of the sheet rows. It also has the following methods:
* Load(sheetId, GetSheetOptions) - Gets sheet info via api. GetSheetOptions controls what rows are loaded (nil=all rows).
* MatchSheet(baseSheet) - Compares cols(id,name) of this instance to a base instance. Returns true/false.  
    Note - the baseSheet instance of SheetInfo would typically be loaded using the Restore(filePath) method.
* Show(...rowLimit) - Displays id, name, cols(id,name,type), rows (limited to rowLimit)
* AddRow(newCells, ...locked) - Adds row, using newCells, to .NewRows slice
* UploadNewRows(rowLocation, rowLevelField) - Uploads .NewRows via API. Use optional rowLevelField for parent/child sets.
* Store(filePath) - save SheetInfo instance as json encrypted file
* Restore(filePath) - reload SheetInfo instance from json encrypted file

## NewCell Type
Used when adding or updating rows. It uses column name rather than column id to identify which column. The AddRow/UpdateRow funcs and SheetInfo.AddRow/UpdateRow methods automatically convert ColName to column id using SheetInfo.ColumnsByName map.
```
type NewCell struct {
	ColName   string
	Formula   string      // only formula or value can be loaded
	Value     interface{} // if hyperlink, value is what's displayed in cell
	Hyperlink *Hyperlink
}
```

## RowValues Func
Makes it easy to reference cell values on a row. It returns returns a map where the key is columnName and cell value is string type. 

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
To update multiple rows using 1 API call. Use SheetInfo.UpdateRow and UploadeUpdateRow methods.
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

## A Few Go Notes
Slices and Maps, if declared but not initialized (using make or initial values), have value = nil.  
If "len" or "range" are used with nil slice or map, it is treated as having zero entries and works properly.
Struct and bool types do not have a "zero" value, so pointers may be used, where nil means no value.  