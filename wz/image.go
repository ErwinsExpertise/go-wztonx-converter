package wz

type WZImage struct {
	*WZSimpleNode // heh
	Properties    *WZProperty
	Parsed        bool
	parseFuncInfo func()
	
	// For thread-safe parallel parsing
	parseFile   *WZFileBlob
	parseOffset int64
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

// ParseWithCopy creates a thread-safe copy of the file and parses with it
// This allows parallel parsing without file position corruption
func (m *WZImage) ParseWithCopy() {
	if m.Parsed {
		return
	}
	
	if m.parseFile == nil {
		// Fallback to original method if parseFile not set
		m.parseFuncInfo()
		return
	}
	
	// Create a thread-safe copy of the file blob for this goroutine
	fileCopy := m.parseFile.Copy()
	m.Parse(fileCopy, m.parseOffset)
}
