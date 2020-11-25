package smartsheet

// InitRow returns new instance of Row.
// If optional parm rowId specified, row.Id field is loaded.
// Row.Cells is also initialized.
func InitRow(rowId ...int64) Row {
	var newRow Row
	if len(rowId) > 0 {
		newRow.Id = rowId[0]
	}
	newRow.Cells = make([]Cell, 0, 50)
	return newRow
}

// CreateLocationMap accepts struct type RowLocation and returns a map.
// Map keys match api location specifier values.
// Fields with value of 0 or false are not included in the map.
func CreateLocationMap(location *RowLocation) map[string]interface{} {

	locMap := make(map[string]interface{})

	if location.ToTop {
		locMap["toTop"] = true
	}
	if location.ToBottom {
		locMap["toBottom"] = true
	}
	if location.ParentId != 0 {
		locMap["parentId"] = location.ParentId
	}
	if location.SiblingId != 0 {
		locMap["siblingId"] = location.SiblingId
	}
	if location.AboveSibling {
		locMap["above"] = true
	}
	if location.Indent != 0 {
		locMap["indent"] = 1
	}
	if location.Outdent != 0 {
		locMap["outdent"] = 1
	}
	return locMap
}
