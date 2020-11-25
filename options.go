package smartsheet

import "time"

// RowLocation indicates where a row should be added or moved to.
type RowLocation struct {
	ParentId, SiblingId           int64 // 0 indicates no parent or sibling, only one can be used
	ToTop, ToBottom, AboveSibling bool  // only one should be true, ToBottom is default when adding rows to sheet without parent
	Indent, Outdent               int   // to activate, load either with value of 1
}

// CopyOptions is used by CopyRows to indicate what elements (in addition to cells) are copied to the destination sheet.
type CopyOptions struct {
	All, Attachments, Children, Discussions bool // specify All or any mix of other options
}

// MoveOptions is used by MoveRows to indicate what elements (in addition to cells) are copied to the destination sheet.
type MoveOptions struct {
	Attachments, Discussions bool // Child rows are always moved
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

// NoRows is a convenience value when requesting no rows be returned by SheetInfo.Load().
var NoRows = &GetSheetOptions{RowIds: []int64{0}}
