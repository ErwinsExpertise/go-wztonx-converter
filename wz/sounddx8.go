package wz

type WZSoundDX8 struct {
	*WZImageObject

	Playtime   int32
	HeaderData []byte
	SoundData  []byte
}

func NewWZSoundDX8(name string, parent *WZSimpleNode) *WZSoundDX8 {
	node := new(WZSoundDX8)
	node.WZImageObject = NewWZImageObject(name, parent)
	return node
}

func (m *WZSoundDX8) Parse(file *WZFileBlob, offset int64) {
	if file.Debug {
		m.debug(file, "> WZSoundDX8::Parse")
		defer func() { m.debug(file, "< WZSoundDX8::Parse") }()
	}

	file.skip(1) // Version number?

	dataLen := file.readWZInt()
	m.Playtime = file.readWZInt()

	m.HeaderData = file.readBytes(82)

	m.SoundData = file.readBytes(dataLen)
}
