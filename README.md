# Smartsheet Go Mini-SDK

Status as of 11/27/2020 > Passing All Tests in sheetinfo_test, row_test, smartsheet_test

Tools for interacting with the Smartsheet API using the Go language.

Install & Use
```
go get github.com/txjmp/smartsheet
import smar "github.com/txjmp/smartsheet"
sheet := new(smar.SheetInfo)
```
This package is not likely to ever implement all Smartsheet API features. To facilitate adding your own features, see CODEINFO.md. It attempts to explain how the code is organized and patterns used.  
[CODEINFO](https://github.com/txjmp/smartsheet/blob/main/CODEINFO.md)

No 3rd party packages are needed.

The SheetInfo type contains most of the information about a sheet including definitions (column ids/names/types) and data (rows). It also has methods for interacting with a sheet.

GoDoc documentation provides complete type and func details. This document is intended to explain functionality in a concise and easy to understand format.  
[GODOC](https://godoc.org/github.com/txjmp/smartsheet)

**To use this package, global var Token must be set with your access token**.
```  
Token = "Bearer youraccesstoken"
```

## Examples  ( also see _test files )
  
### Create an instance of SheetInfo, Load It Via the API, Store It, and Show It
```
sheetX := new(SheetInfo)

sheetX.Load(sheetXId, nil)  // sheetXId contains the sheet id, nil indicates no GET sheet options

sheetX.Store("sheets/sheetx.json")  // store a copy of the SheetInfo as a json encrypted file

sheetX.Show(5) // display sheet id, name, column names/types and rows (limit to 5 rows)
```

### Verify SheetInfo Columns & Types Match a Base Version
```
sheetXBase := new(SheetInfo)

sheetXBase.Restore("sheets/sheetx_base.json")  // restore from json encrypted file

matched := SheetX.Match(sheetXBase) // returns true or false and displays differences
```

### Get Sheet Options - Controls What is Returned by API Get Sheet 
```
type GetSheetOptions struct {
	RowIds            []int64   // include only specific rows
	RowsModifiedSince time.Time // include only rows modified since specific time
	RowsModifiedMins  int       // include only rows where modified-time within x minutes before current time
	ColumnNames       []string  // used by sheetInfo.Load to get columnIds, not used by GetSheet func
	ColumnIds         []int64   // include only specified columns
}

rowIds := []int64{6840477608372100, 23866684047796654, 684898239820023}
options := GetSheetOptions{ 
    RowIds: rowIds,
    ColumnNames: []string{"Customer", "Location"},
}
sheetX.Load(sheetXId, &options) // & passes pointer to options

if options is nil, all rows and columns returned.
```

### Add Rows With Parent & Child
New rows are first added to SheetInfo.NewRows slice using AddRow method.
UploadNewRows adds NewRows to the sheet via API.
```
var newRow Row
// -- Add Parent Row -----------------------------------------
newRow = InitRow()
newRow.Cells = []Cell{
	{ColName: "Step", Value: "Start" },
	{ColName: "Level", Value: "0"},  // parent indicator
}
err = sheet.AddRow(newRow)

// -- Add Child Row -----------------------------------------
linkedDoc := &Hyperlink{Url:"https://..."}  // linkedDoc is pointer
newRow = InitRow()
newRow.Cells = []Cell{
	{ColName: "Step", Value: "Start" },
	{ColName: "Level", Value: "1"},  // child indicator
	{ColName: "Phase", Value: "Design"},
	{ColName: "Doc", Value: "Linked Doc", Hyperlink: linkedDoc},
	{ColName: "Rating", Value: 92.7},
}
err = sheet.AddRow(newRow)

NOTE - Value can be of type string, int, int64, float64, bool

// -- Upload Rows -------------------------------------------
rowLevelField := "Level" // used to set parent/child relationship  (parent-0, child-1)
response, err := sheet.UploadNewRows(nil, rowLevelField)  // nil indicates to use default rowLocation (bottom of sheet)
```

### Discussion of Adding Rows
---
UploadNewRows method has 2 parameters, rowLocation and rowLevelField.
RowLocation determines where the row(s) are placed. If nil, rows are added to the bottom of the sheet.  

RowLevelField is optional. If not included, then rows are added based only on RowLocation. If included, it indicates the column name that determines whether a row is a parent or child. Values of "0" parent and "1" child are required.

Rows must be added in the order of parent-child-child, parent-child-child, etc.

The UploadNewRows method performs 1 api call to add all rows and an additional call (using rowIds from 1st call results) for each set of children (setting the parentId).

---  

### Row Location Type - Indicates Where row(s) Should be Added or Moved To
Used by UploadNewRows and UploadUpdateRows. Code must use viable options. For example setting both ToBottom and ToTop true is not viable.
An understanding of the API Row Location rules is recommended.
```
type RowLocation struct {
	ParentId  	int64
	SiblingId 	int64
	ToTop 		bool
	ToBottom 	bool
	AboveSibling bool
	Indent 		int
	Outdent 	int
}
```
### Update Rows
Updated rows are first added to SheetInfo.UpdateRows slice using UpdateRow method.
UploadUpdateRows updates sheet rows via API. 
```
updtRow := InitRow(rowId)  // updtRow.Id will be loaded with rowId
updtRow.Locked = &IsTrue   // Locked is pointer type (allows use of nil to indicate no value)
updtRow.Cells = []Cell{
	{ColName: "DueDate", Value: "2020-12-22"},
	{ColName: "Status", Value: "Pending"},
}
sheet.UpdateRow(updtRow)
location := RowLocation{ToTop:true}
response, err := sheet.UploadUpdateRows(&location)
```

### Referencing Row Values
Func RowValues returns the cell values of a row as a map[string]string. Key of each map entry is column name. Value of each map entry is a string representation of the value. Numbers do not contain formatting such as $ and commas. Hyperlink values return the url. Multi value cells return all values concatenated together. To access all cell information such as cell link values, use CellInfo func.
```
sheetX.Load(sheetXId, nil)  // loads all rows
for i, row := range sheetX.Rows {
	fmt.Println("Row Id", row.Id)
	vals := RowValues(sheetX, row)   // vals is type map[string]string
	fmt.Println(i, vals["Customer"], " - ", vals["Address"])  // ex. 1 TopButton - 1200 Canton Road
}
```

### CellInfo Func
Convenient way to reference a particular cell. Provides access to all cell attributes.
```
cell := CellInfo(sheetX, row, "ColumnName")  // cell is type Cell
```

### Copy & Move Rows
CopyOptions is used by CopyRows to indicate what elements (in addition to cells) are copied to the destination sheet. If nil, none are copied.
```
type CopyOptions struct {
	All, Attachments, Children, Discussions bool // specify All or any mix of other options
}
options := CopyOptions{All:true}
rowIds := []int64{rowId1, rowId2}
err := CopyRows(fromSheetId, rowIds, toSheetId, &options)

// move rows, children of parent rows are automatically copied
err := MoveRows(fromSheetId, rowIds, toSheetId, &options)
```

### GetRow Func
Returns a single row via API.
```
row, err := GetRow(sheetId, rowId)
```
### AddRow, UpdateRow Funcs
Add or Update 1 row via API. See AddRow, UpdateRow SheetInfo discussion above for details.
```
response, err := AddRow(sheetX, newRow, &location)
response, err := UpdateRow(sheetX, updtRow, &location)
```

### SetParentId Func
Sets the parent id for child row(s). If a single child row, it will be 1st child of parent, unless optional toBottom is true.
```
err := SetParentId(sheetId, parentId, childIds)  // childIds []int64
```

### Attach File or URL To Row
```
err := AttachFileToRow(sheetId, rowId, filePath)
err := AttachUrlToRow(sheetId, rowId, attachmentName, attachmentType, linkUrl)
```

### Other Features
```
Create,List CrossSheetReferences (required for Cross Sheet Formulas)
Create,Enable,Get,Delete Webhooks
```

### Types
```
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
type Column struct {
	Id      int64    `json:"id"`
	Index   int      `json:"index"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Primary bool     `json:"primary"`
	Options []string `json:"options"`
}
type Cell struct {
	ColName         string      `json:"-"`   // not used by API
	ColumnId        int64       `json:"columnId"`
	Formula         string      `json:"formula,omitempty"`
	Hyperlink       *Hyperlink  `json:"hyperlink,omitempty"`
	LinkInFromCell  *CellLink   `json:"linkInFromCell,omitempty"`
	LinksOutToCells []CellLink  `json:"linksOutToCells,omitempty"`
	Value           interface{} `json:"value,omitempty"`
}
type Row struct {
	Id     int64  `json:"id"`
	Cells  []Cell `json:"cells"`
	Locked *bool  `json:"locked"` // when updating rows: nil-nochange, false-unlock, true-lock
}
type Hyperlink struct {
	Reportid int64  `json:"reportId"`
	Sheetid  int64  `json:"sheetId"`
	Url      string `json:"url"`
}
type CellLink struct {
	ColumnId int64  `json:"reportId"`
	RowId    int64  `json:"rowId"`
	SheetId  int64  `json:"sheetId"`
	Status   string `json:"status"`
}
type CrossSheetReference struct {
	Name          string `json:"name"`
	SourceSheetId int64  `json:"sourceSheetId"`
	StartRowId    int64  `json:"startRowId,omitempty"` // omit for all rows
	EndRowId      int64  `json:"endRowId,omitempty"`   // omit for all rows
	StartColumnId int64  `json:"startColumnId"`
	EndColumnId   int64  `json:"endColumnId"`
}
type AddUpdtRowsResponse struct {
	Message    string `json:"message"`    // ex. "SUCCESS"
	ResultCode int    `json:"resultCode"` // ex. 0
	Result     []Row  `json:"result"`
}

```