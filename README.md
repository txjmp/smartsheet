# Smartsheet Go Mini-SDK

Tools for interacting with the Smartsheet API using the Go language. Functionality currently includes a limited set of features.

No 3rd party packages are needed.

The SheetInfo type contains most of the information about a sheet including definitions (column ids/names/types) and data (rows). It also has methods for interacting with a sheet.

GoDoc documentation will provide complete type and func details. This document is intended to explain functionality in a more concise and easy to understand format.

## Examples

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
type NewCell struct {
	ColumnName string
	Formula    string      // only formula or value can be loaded
	Value      interface{} // if hyperlink, value is what's displayed in cell
	Hyperlink  *Hyperlink
}

// -- Add Parent Row -----------------------------------------
newRow := []NewCell{
	{ColName: "Step", Value: "Start" },
	{ColName: "Level", Value: "0"},  // parent indicator
}
err = sheet.AddRow(newRow)

// -- Add Child Row -----------------------------------------
linkedDoc := &Hyperlink{Url:"https://..."}  // linkedDoc is pointer
newRow := []NewCell{
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

rowLocation := RowLocation{ToTop:true}
sheetX.UploadNewRows( &rowLocation )

```
### Update Rows
Updated rows are first added to SheetInfo.UpdateRows slice using UpdateRow method.
UploadUpdateRows updates sheet rows via API. 
```
	updateRow := []NewCell{
		{ColName: "DueDate", Value: "2020-12-22"},
		{ColName: "Status", Value: "Pending"},
	}
	locked := true  // indicates this row should be locked, use false to unlock
	sheet.UpdateRow(rowId, updateRow, locked)  // if locked parm omitted, lock status not changed
	rowLocation := new(RowLocation)
	rowLocation.ToTop = true
	response, err := sheet.UploadUpdateRows(rowLocation)
```

### Referencing Row Values
Row type contains RowId, []Cell, Locked indicator.
Func RowValues returns the cell values of a row as a map[string]string. Key of each map entry is column name. Value of each map entry is a string representation of the value. Numbers do not contain formatting such as $ and commas. Hyperlink values return the url. Multi value cells return all values concatenated together. To access all cell information such as cell link values, use CellInfo func.
```
sheetX.Load(sheetXId, nil)
var rowCells map[string]string
for i, row := range sheetX.Rows {
	fmt.Println("Row Id", row.Id)
	rowCells = RowValues(sheetX, row)
	fmt.Println(i, rowCells["Customer"], " - ", rowCells["Address"])  // ex. 1 TopButton - 1200 Canton Road
}
```

### CellInfo Func
Provides access to additional cell values such as Cell Link, Formula, Hyperlink.
```
	cellData := CellInfo(sheetX, row, "ColumnName")
	cellData.LinkInFromCell is type CellLink.
	type CellLink struct {
		ColumnId int64  `json:"reportId"`
		RowId    int64  `json:"rowId"`
		SheetId  int64  `json:"sheetId"`
		Status   string `json:"status"`
	}
```

### Copy & Move Rows
```
// CopyOptions is used by CopyRows to indicate what elements (in addition to cells) are copied to the destination sheet. If nil, none are copied.
type CopyOptions struct {
	All, Attachments, Children, Discussions bool // specify All or any mix of other options
}
options := CopyOptions{All:true}
rowIds := []int64{877464703340856, 88023437740234870}
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
Add or Update 1 row via API.
```
// sheetX *SheetInfo
// newCells []NewCell
// rowLocation *RowLocation  (nil to use default or no change)
// locked bool (optional, omit to use default or no change)
result, err := AddRow(sheetX, newCells, rowLocation, locked)
result, err := UpdateRow(sheetX, rowId, newCells, rowLocation, locked)
```

### SetParentId Func
Sets the parent id for child row(s). If a single child row, it will be 1st child of parent, unless optional toBottom is true.
```
// childIds []int64
err := SetParentId(sheetId, parentId, childIds)
```

### Attach File or URL To Row
```
err := AttachFileToRow(sheetId, rowId, filePath)
err := AttachUrlToRow(sheetId, rowId, fileName, attachmentType, linkUrl)
```

### Other Features
```
Create,List CrossSheetReferences (required for Cross Sheet Formulas)
Create,Enable,Get,Delete Webhooks
Code to process Webhook requests using Go built-in web server (not in this package)
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
type NewCell struct {
	ColumnName string
	Formula    string      // only formula or value can be loaded
	Value      interface{} // if hyperlink, value is what's displayed in cell
	Hyperlink  *Hyperlink
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