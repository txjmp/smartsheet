package smartsheet

import (
	"io/ioutil"
	"strings"
	"testing"
)

func Test_Email(t *testing.T) {
	var sheetId int64 = 1849449510135684

	var err error

	tkn, _ := ioutil.ReadFile("token.txt")
	Token = strings.TrimSpace(string(tkn))

	TraceOn = true
	DebugOn = true

	sheet := new(SheetInfo)
	sheet.Load(sheetId, nil)

	recipients := []EmailRecipient{
		{"email": "txjmp19@gmail.com"},
	}
	addressColId := sheet.ColumnsByName["Address"].Id
	orderNoColId := sheet.ColumnsByName["OrderNo"].Id

	emailParms := EmailRowsObj{
		SendTo:    recipients,
		Subject:   "Test Email Rows",
		Message:   "Rows From Sheet Test 1",
		RowIds:    []int64{sheet.Rows[1].Id, sheet.Rows[2].Id},
		ColumnIds: []int64{addressColId, orderNoColId},
	}
	err = EmailRows(sheetId, emailParms)
	if err != nil {
		t.Fatal("Test_Email EmailRows Failed", err)
	}
}
