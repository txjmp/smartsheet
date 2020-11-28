package smartsheet

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func Test_Smartsheet(t *testing.T) {
	var sheet1Id int64 = 1849449510135684
	var sheet2Id int64 = 8094487248430980

	var err error

	tkn, _ := ioutil.ReadFile("token.txt")
	Token = strings.TrimSpace(string(tkn))

	TraceOn = true
	DebugOn = true

	// === CREATE SHEET FILE ===========================================
	err = GetSheetAs(sheet1Id, "/home/jay/sheets/sheet1.xlsx", EXCEL)
	if err != nil {
		t.Fatal("Test_Smartsheet GetSheetRows Failed", err)
	}
	err = GetSheetAs(sheet1Id, "/home/jay/sheets/sheet1.pdf", PDF, "WIDE")
	if err != nil {
		t.Fatal("Test_Smartsheet GetSheetRows Failed", err)
	}
	err = GetSheetAs(sheet1Id, "/home/jay/sheets/sheet1.csv", CSV)
	if err != nil {
		t.Fatal("Test_Smartsheet GetSheetRows Failed", err)
	}
	// === CELLINFO ===========================================
	sheet1 := new(SheetInfo)
	//sheet1.Load(sheet1Id, nil)
	//sheet1.Store("/home/jay/sheets/sheet1.json")
	sheet1.Restore("/home/jay/sheets/sheet1.json")

	var orderNoCell *Cell
	colName := "Hyperlink"
	orderNoCell = CellInfo(sheet1, sheet1.Rows[1], colName)
	if orderNoCell == nil {
		t.Fatal("Test_Smartsheet CellInfo Failed", err)
	}
	fmt.Printf("%+v\n", orderNoCell)
	fmt.Println(orderNoCell.Hyperlink)

	// === COPY / MOVE ROWs ===========================================
	fromSheetId := sheet1Id
	toSheetId := sheet2Id
	rowIds := []int64{6935334990440324, 4172552448567172}
	options := CopyOptions{Discussions: true}
	err = CopyRows(fromSheetId, rowIds, toSheetId, &options)
	if err != nil {
		t.Fatal("Test_Smartsheet CopyRows Failed", err)
	}
	moveOptions := MoveOptions{Discussions: true}
	err = MoveRows(fromSheetId, rowIds, toSheetId, &moveOptions)
	if err != nil {
		t.Fatal("Test_Smartsheet MoveRows Failed", err)
	}

	// === ATTACH FILE TO ROW =================================
	err = AttachFileToRow(sheet1Id, 6840477608372100, "/home/jay/test/attachment.txt")
	if err != nil {
		t.Fatal("Test_Smartsheet AttachFileToRow Failed", err)
	}

	// === ATTACH URL TO ROW ==================================
	err = AttachUrlToRow(sheet1Id, 6840477608372100, "parrot.jpeg", LINK, "https://unsplash.com/photos/QxHJ9lkXYNk")
	if err != nil {
		t.Fatal("Test_Smartsheet AttachFileToRow Failed", err)
	}
}
