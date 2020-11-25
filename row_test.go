package smartsheet

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func Test_Row(t *testing.T) {
	var sheet1Id int64 = 1849449510135684
	var sheet2Id int64 = 8094487248430980

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

	// === COPY ROW ===========================================
	rowIds := []int64{response1.Result.Id}
	options := CopyOptions{Attachments: true, Discussions: true}
	err = CopyRows(sheet1Id, rowIds, sheet2Id, &options)
	if err != nil {
		t.Fatal("Test_Row CopyRows Failed", err)
	}

	// === MOVE ROW ===========================================
	moveOptions := MoveOptions{Attachments: true}
	err = MoveRows(sheet1Id, rowIds, sheet2Id, &moveOptions)
	if err != nil {
		t.Fatal("Test_Row MoveRows Failed", err)
	}

	// === DELETE ROWS ========================================
	rowIds = []int64{2858796753938308, 7803652731365252}
	err = DeleteRows(sheet1Id, rowIds...) // ... splits slice into individual elements
	if err != nil {
		t.Fatal("Test_Row DeleteRows Failed", err)
	}
}
