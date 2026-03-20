package repository

func normalizePagination(params ListParams) (limit int, offset int) {
	limit = params.Limit
	offset = params.Offset

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
