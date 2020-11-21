package smartsheet

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

var TestRow_SheetId int64 = 1849449510135684

func Test_Row(t *testing.T) {
	var err error

	tkn, _ := ioutil.ReadFile("token.txt")
	Token = strings.TrimSpace(string(tkn))

	TraceOn = true
	DebugOn = true

	sheet := new(SheetInfo)
	sheet.Restore("/home/jay/sheets/test1_base.json")

	// === ADD ROW ===========================================
	newRow := InitRow()
	newRow.Cells = []Cell{
		{ColName: "OrderNo", Value: "3400"},
		{ColName: "Util", Value: "Water"},
		{ColName: "Amt", Value: 120.40},
	}
	location := RowLocation{ToTop: true}
	response1, err := AddRow(sheet, newRow, &location)
	if err != nil {
		t.Fatal("Test_Row AddRow Failed", err)
	}
	fmt.Println(response1)

	// === UPDATE ROW ===========================================
	rowId := response1.Result.Id
	updtRow := InitRow(rowId) // updating row just added
	updtRow.Cells = []Cell{
		{ColName: "Complete", Value: true},
	}
	_, err = UpdateRow(sheet, updtRow, nil)
	if err != nil {
		t.Fatal("Test_Row UpdateRow Failed", err)
	}
}
