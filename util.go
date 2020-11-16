package smartsheet

// CreateLocationMap accepts struct type RowLocation and returns a map.
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
