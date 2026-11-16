package books

const (
	wanna_read = "wanna_read"
	read       = "read"
	reading    = "reading"
)

func statusForPage(currentPage int32, totalPages int32) string {
	if currentPage == 0 {
		return wanna_read
	}
	if currentPage == totalPages {
		return read
	}
	return reading
}
