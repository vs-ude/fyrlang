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

// Join ...
func (l LocationRange) Join(l2 LocationRange) LocationRange {
	if l.IsNull() {
		return l2
	}
	if l2.IsNull() {
		return l
	}
	return LocationRange{From: l.From, To: l2.To}
}

// IsNull ...
func (l LocationRange) IsNull() bool {
	return l.From == 0
}

// IsEqualLine ...
func IsEqualLine(l1, l2 LocationRange) bool {
	f1 := uint64(l1.From) >> 48
	line1 := int((uint64(l1.From) & 0xffff00000000) >> 32)
	f2 := uint64(l2.From) >> 48
	line2 := int((uint64(l2.From) & 0xffff00000000) >> 32)

	return f1 == f2 && line1 == line2
}

// StripPosition ...
func StripPosition(l LocationRange) LocationRange {
	l.From = Location(uint64(l.From) &^ 0xffffffff)
	l.To = Location(uint64(l.To) &^ 0xffffffff)
	return l
}
