package smartsheet

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

var Test1Id int64 = 1849449510135684
var Test2Id int64 = 7955014627944324
var AllTypesId int64 = 3841586741176196

type rec struct {
	Address, OrderNo, DueDate string
}

var testData = []rec{
	{"400 Ringo", "488", "2020-10-10"},
	{"500 Delta", "522", "2020-11-22"},
	{"600 Micky", "699", "2020-12-31"},
}

func Test_SheetInfo(t *testing.T) {
	var err error
	setToken()

	//TraceOn = true
	//DebugOn = true

	if err = series1(); err != nil {
		t.Error("Test_SheetInfo series1 Failed", err)
	}
	if err = series2(); err != nil {
		t.Error("Test_SheetInfo series2 Failed", err)
	}
	if err = series3(); err != nil {
		t.Error("Test_SheetInfo series3 Failed", err)
	}
}

func setToken() {
	tkn, _ := ioutil.ReadFile("token.txt")
	Token = strings.TrimSpace(string(tkn))
}

// Load, Store, Restore, Match
func series1() error {
	var err error
	test1 := new(SheetInfo)
	if err = test1.Load(Test1Id, NoRows); err != nil {
		return err
	}
	if err = test1.Store("/home/jay/sheets/test1_base.json"); err != nil {
		return err
	}
	test1Base := new(SheetInfo)
	if err = test1Base.Restore("/home/jay/sheets/test1_base.json"); err != nil {
		return err
	}
	if matched := test1.MatchSheet(test1Base); !matched {
		return errors.New("Match Failed")
	}
	return nil
}

// Add Rows, Update Rows, Show
func series2() error {
	var err error
	sheetId := Test2Id

	sheet := new(SheetInfo)
	if err = sheet.Load(sheetId, NoRows); err != nil {
		return err
	}
	// === ADD ROWS ==================================================================

	var newRow Row
	for _, data := range testData {
		// -- Add Parent Row -----------------------------------------
		newRow = InitRow()
		newRow.Cells = []Cell{
			{ColName: "Address", Value: data.Address},
			{ColName: "Level", Value: "0"},
		}
		sheet.AddRow(newRow)

		// -- Add Child Rows -----------------------------------------
		newRow = InitRow()
		newRow.Cells = []Cell{
			{ColName: "Address", Value: data.Address},
			{ColName: "Level", Value: "1"},
			{ColName: "OrderNo", Value: data.OrderNo},
			{ColName: "DueDate", Value: data.DueDate},
			{ColName: "Util", Value: "Elec"},
			{ColName: "Hyperlink", Value: "cheepcode", Hyperlink: &Hyperlink{Url: "https://cheepcode.com"}},
			{ColName: "Amt", Value: 74.20},
			{ColName: "Complete", Value: false},
		}
		sheet.AddRow(newRow)

		newRow = InitRow()
		newRow.Cells = []Cell{
			{ColName: "Address", Value: data.Address},
			{ColName: "Level", Value: "1"},
			{ColName: "OrderNo", Value: data.OrderNo},
			{ColName: "DueDate", Value: data.DueDate},
			{ColName: "Util", Value: "Water"},
			{ColName: "Amt", Value: 33.75},
			{ColName: "Complete", Value: true},
		}
		sheet.AddRow(newRow)
	}
	// -- Upload Rows -----
	rowLevelField := "Level"                                 // used to set parent/child relationship  (parent-0, child-1)
	response, err := sheet.UploadNewRows(nil, rowLevelField) // use default row location
	if err != nil {
		fmt.Println("UploadNewRows failed", err)
		return err
	}
	if len(response.Result) != 9 {
		fmt.Println("rows loaded", len(response.Result))
		return errors.New("Wrong Number of Rows Added")
	}

	// === UPDATE ROWS ==================================================================

	sheet.Load(sheetId, nil)

	// If DueDate column value is before today, set Status to red
	today := time.Now().Format(DateFormat) // "yyyy-mm-dd"
	for _, row := range sheet.Rows {
		vals := RowValues(sheet, row)
		if vals["Level"] == "1" { // child row
			if vals["DueDate"] < today {
				updtRow := InitRow(row.Id)
				updtRow.Cells = []Cell{
					{ColName: "Status", Value: "Red"},
				}
				// or
				//updtRow.Cells = append(updtRow.Cells, Cell{ColName: "Status", Value::"Red"})
				sheet.UpdateRow(updtRow)
			}
		}
	}
	_, err = sheet.UploadUpdateRows(nil)

	// === SHOW ROWS ==================================================================

	sheet.Load(sheetId, nil)
	sheet.Show()

	return err
}

// Use GetSheetOptions to limit rows returned
func series3() error {
	var err error

	sheetId := Test2Id
	sheet := new(SheetInfo)
	sheet.Load(sheetId, NoRows)

	getOptions := GetSheetOptions{
		RowsModifiedSince: time.Now().Add(-20 * time.Minute),
		ColumnNames:       []string{"Address", "Status", "DueDate"},
	}
	err = sheet.Load(sheetId, &getOptions)
	sheet.Show()
	return err
}
