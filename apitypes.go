// apitypes.go contains types that match objects sent & received directly by Smartsheet API

package smartsheet

// Hyperlink cell field stores link information.
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

type Column struct {
	Id      int64    `json:"id"`
	Index   int      `json:"index"`
	Title   string   `json:"title"`
	Type    string   `json:"type"`
	Primary bool     `json:"primary"`
	Options []string `json:"options"`
}

// Cell is used in both api responses and api requests.
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

// Row is used in api responses but not api requests.
// It is also used when adding and updating rows. See SheetInfo.NewRows, UpdateRows.
// For row add/update requests, a dynamic object is created using map[string]interface{} (includes row location fields).
type Row struct {
	Id     int64  `json:"id"`
	Cells  []Cell `json:"cells"`
	Locked *bool  `json:"locked"` // when updating rows: nil-nochange, false-unlock, true-lock
}

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

// Response object for Adding or Updating Rows
type AddUpdtRowsResponse struct {
	Message    string `json:"message"`    // ex. "SUCCESS"
	ResultCode int    `json:"resultCode"` // ex. 0
	Result     []Row  `json:"result"`
}
