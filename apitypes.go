// apitypes.go contains types that match objects sent & received directly by Smartsheet API

package smartsheet

// Hyperlink is used in Row Cells to store hyperlink information.
// The link can be to a URL, Sheet, or Report.
type Hyperlink struct {
	Reportid int64  `json:"reportId,omitempty"`
	Sheetid  int64  `json:"sheetId,omitempty"`
	Url      string `json:"url,omitempty"`
}

// CellLink identifies location of linked value.
type CellLink struct {
	ColumnId int64  `json:"reportId"`
	RowId    int64  `json:"rowId"`
	SheetId  int64  `json:"sheetId"`
	Status   string `json:"status"`
}

// Column contains values from API Get Sheet.
type Column struct {
	Id      int64    `json:"id"`
	Index   int      `json:"index"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Primary bool     `json:"primary"`
	Options []string `json:"options"`
}

// Cell contains cell values.
// It is used in both api responses and requests.
// It is also used when adding and updating rows. See SheetInfo.AddRow, UpdateRow.
type Cell struct {
	ColName         string      `json:"-"` // not used by API
	ColumnId        int64       `json:"columnId"`
	Formula         string      `json:"formula,omitempty"`
	Hyperlink       *Hyperlink  `json:"hyperlink,omitempty"`
	LinkInFromCell  *CellLink   `json:"linkInFromCell,omitempty"`
	LinksOutToCells []CellLink  `json:"linksOutToCells,omitempty"`
	Value           interface{} `json:"value,omitempty"`
}

// Row is used in api responses but not directly in api requests.
// It is used when adding and updating rows. See SheetInfo.AddRow, UpdateRow.
type Row struct {
	Id     int64  `json:"id"`
	Cells  []Cell `json:"cells"`
	Locked *bool  `json:"locked"` // when updating rows: nil-nochange, false-unlock, true-lock
}

// Sheet is the api response for GetSheet.
// It is not typically directly used by other processes.
// See SheetInfo type, which contains these Sheet values.
type Sheet struct {
	Id            int64  `json:"id"`
	Name          string `json:"name"`
	TotalRowCount int    `json:"totalRowCount"`
	Workspace     struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"workspace"`
	Permalink  string   `json:"permalink"`
	CreatedAt  string   `json:"createdAt"`
	ModifiedAt string   `json:"modifiedAt"`
	Columns    []Column `json:"columns"`
	Rows       []Row    `json:"rows"`
}

type CrossSheetReference struct {
	Name          string `json:"name"`
	SourceSheetId int64  `json:"sourceSheetId"`
	StartRowId    int64  `json:"startRowId,omitempty"` // omit for all rows
	EndRowId      int64  `json:"endRowId,omitempty"`   // omit for all rows
	StartColumnId int64  `json:"startColumnId"`
	EndColumnId   int64  `json:"endColumnId"`
}

// AddUpdtRowsResponse is api response object when adding mutiple rows or updating 1 or more rows.
type AddUpdtRowsResponse struct {
	Message    string `json:"message"`    // ex. "SUCCESS"
	ResultCode int    `json:"resultCode"` // ex. 0
	Result     []Row  `json:"result"`
}

// Add1RowResponse is api response object when adding 1 row.
type Add1RowResponse struct {
	Message    string `json:"message"`    // ex. "SUCCESS"
	ResultCode int    `json:"resultCode"` // ex. 0
	Result     Row    `json:"result"`
}
