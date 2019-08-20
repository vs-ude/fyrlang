package errlog

// Location ...
type Location int64

// LocationRange ...
type LocationRange struct {
	From Location
	To   Location
}

// LocationMap ...
type LocationMap struct {
	files []*SourceFile
}

// SourceFile ...
type SourceFile struct {
	Name string
}

// NewSourceFile ...
func NewSourceFile(name string) *SourceFile {
	return &SourceFile{Name: name}
}

// EncodeLocation ...
func EncodeLocation(file int, line int, pos int) Location {
	return Location((uint64(file) << 48) | (uint64(line<<32) | (uint64(pos))))
}

// EncodeLocationRange ...
func EncodeLocationRange(file int, fromLine int, fromPos int, toLine int, toPos int) LocationRange {
	from := Location((uint64(file) << 48) | (uint64(fromLine<<32) | (uint64(fromPos))))
	to := Location((uint64(file) << 48) | (uint64(toLine<<32) | (uint64(toPos))))
	return LocationRange{From: from, To: to}
}

// NewLocationMap ...
func NewLocationMap() *LocationMap {
	return &LocationMap{}
}

// AddFile ...
func (l *LocationMap) AddFile(f *SourceFile) int {
	l.files = append(l.files, f)
	return len(l.files) - 1
}

// Decode ...
func (l *LocationMap) Decode(loc Location) (*SourceFile, int, int) {
	file := l.files[uint64(loc)>>48]
	line := int((uint64(loc) & 0xffff00000000) >> 32)
	pos := int(uint64(loc) & 0xffffffff)
	return file, line, pos
}
