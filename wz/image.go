package wz

type WZImage struct {
	*WZSimpleNode // heh
	Properties    *WZProperty
	Parsed        bool
	parseFuncInfo func()
	parseFile     *WZFileBlob // Original file reference for thread-safe parsing
	parseOffset   int64       // Original offset for thread-safe parsing
}

func NewWZImage(name string, parent *WZSimpleNode) *WZImage {
	node := new(WZImage)
	node.WZSimpleNode = NewWZSimpleNode(name, parent)
	node.Parsed = false
	return node
}

func (m *WZImage) Parse(file *WZFileBlob, offset int64) {
	if m.Parsed {
		return
	}

	if file.Debug {
		m.debug(file, "> WZImage::Parse")
		defer func() { m.debug(file, "< WZImage::Parse") }()
	}

	file.seek(offset)
	typename := file.readDeDuplicatedWZString(m.GetPath(), offset, true)
	parsedObject := ParseObject(m.Name, typename, m.WZSimpleNode, file, offset)

	objResult, isOK := parsedObject.(*WZProperty)
	if !isOK {
		panic("Expected object to be *WZProperty")
	}

	m.Properties = objResult
	m.Parsed = true
}

func (m *WZImage) StartParse() {
	if m.Parsed {
		return
	}

	m.parseFuncInfo()
}

// ParseWithCopy creates a thread-safe copy of WZFileBlob for parallel parsing
// This prevents bytes.Reader corruption when multiple goroutines parse images concurrently
func (m *WZImage) ParseWithCopy() {
	if m.Parsed {
		return
	}

	// Create a thread-safe copy of the file blob for this goroutine
	if m.parseFile != nil {
		fileCopy := m.parseFile.Copy()
		m.Parse(fileCopy, m.parseOffset)
	} else {
		// Fallback to original method if parseFile not set
		m.parseFuncInfo()
	}
}
