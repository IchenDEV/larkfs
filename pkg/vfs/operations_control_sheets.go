package vfs

func sheetsQueryActionNames() []string {
	return []string{
		"workbook-info",
		"csv-get",
		"cells-get",
		"cells-search",
		"sheet-info",
		"chart-list",
		"cond-format-list",
		"dropdown-get",
		"filter-list",
		"filter-view-list",
		"float-image-list",
		"pivot-list",
		"sparkline-list",
	}
}

func sheetsOpActionNames() []string {
	return []string{
		"workbook-create",
		"workbook-export",
		"batch-update",
		"csv-put",
		"cells-set",
		"cells-batch-clear",
		"cells-batch-set-style",
		"cells-clear",
		"cells-merge",
		"cells-replace",
		"cells-set-image",
		"cells-set-style",
		"cells-unmerge",
		"chart-create",
		"chart-delete",
		"chart-update",
		"cond-format-create",
		"cond-format-delete",
		"cond-format-update",
		"cols-resize",
		"rows-resize",
		"dim-delete",
		"dim-freeze",
		"dim-group",
		"dim-hide",
		"dim-insert",
		"dim-move",
		"dim-ungroup",
		"dim-unhide",
		"dropdown-delete",
		"dropdown-set",
		"dropdown-update",
		"filter-create",
		"filter-delete",
		"filter-update",
		"filter-view-create",
		"filter-view-delete",
		"filter-view-update",
		"float-image-create",
		"float-image-delete",
		"float-image-update",
		"pivot-create",
		"pivot-delete",
		"pivot-update",
		"range-copy",
		"range-fill",
		"range-move",
		"range-sort",
		"sheet-copy",
		"sheet-create",
		"sheet-delete",
		"sheet-hide",
		"sheet-move",
		"sheet-rename",
		"sheet-set-tab-color",
		"sheet-unhide",
		"sparkline-create",
		"sparkline-delete",
		"sparkline-update",
	}
}

func sheetsQuerySpecs() map[string]actionSpec {
	return plusActionSpecs("sheets", sheetsQueryActionNames())
}

func sheetsActionSpecs() map[string]actionSpec {
	return plusActionSpecs("sheets", sheetsOpActionNames())
}
