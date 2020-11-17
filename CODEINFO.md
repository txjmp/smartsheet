# Code Info

The purpose of this document is to describe the organization and techniques used to implement functionality.

## URL Parameters
Sometimes the API wants values passed as query parameters in the URL. The Get, Post, Put, Delete funcs in request.go handle this process. They accept a map[string]string parameter containing the key:value combinations. If a key has multiple values, all values must be joined together, separated by commas. FYI, Go's URL.Values type has an Add method that allows different values to be added to the same key. Unfortunately the Values.Encode method creates a separate key=value(for each value) in the url query string (which Smartsheet does not like).

## Request Data
For most POST and PUT requests, data is placed into the http request body. The Post, Put funcs in request.go handle this process. The calling func passes the data in whatever format the API requires. For example, the following is used to Copy Rows:
```
var reqData struct {
	RowIds  []int64 `json:"rowIds"`
	To      struct {
		SheetId int64 `json:"sheetId"`
	} `json:"to"`
}
reqData.RowIds = rowIds
reqData.To.SheetId = toSheetId
```    

## Typical Process Steps
1. Set call options if needed (ex. RowLocation, GetSheetOptions, CopyOptions)
2. Create URL parameter map if needed (ex. CopyRow options)
3. Build request data if needed (for most Post & Put calls)
4. Set the URL end point for the API call.
5. Call http request builder (Get, Post, Put, Delete)
6. Execute the call

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

## Example
Update Status column of a row in the "Tasks" sheet.
Also change it's location (last child of different parent row), and lock it.
```
var sheetTasks *SheetInfo  // loaded by other code

location := RowLocation{ ParentId:parentRowId, ToBottom:true }  // parentRowId is int64 var

updateCells := make([]Cell,1)  // this example is updating 1 cell in the row

updateCells[0] := NewCell{     // create update cell
    ColName: "Status",
    Value:   "Hold",
}
locked := true
UpdateRow(sheetTasks, rowId, updateCells, &location, locked )
//  constructs request body using sheetInfo, rowId, updateCells, location

```
## Organization

### SheetInfo
SheetInfo is the predominant type for performing tasks.
It contains attributes such as sheetId, sheetName, and column info.
It also contains data (rows).
```
vendorSheet := new(SheetInfo)  // create new instance
vendorSheet.Load(vendorSheetId, options)  // using the API, sheet info is downloaded
vendorSheet.Rows now contains the row data.
```
Th


There are independent functions such as GetRow and methods of the SheetInfo type
