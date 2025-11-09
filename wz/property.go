package wz

type WZProperty struct {
	Properties map[string]*WZVariant
	Order      []string // Preserves insertion order
}

func ParseProperty(parent *WZSimpleNode, file *WZFileBlob, offset int64) *WZProperty {
	if file.Debug {
		parent.debug(file, "> WZProperty::Parse")
		defer func() { parent.debug(file, "< WZProperty::Parse") }()
	}

	file.skip(2) // Unk
	propcount := int(file.readWZInt())

	if file.Debug {
		parent.debug(file, "Properties of ", parent.GetPath(), ": ", propcount)
	}

	result := &WZProperty{
		Properties: make(map[string]*WZVariant),
		Order:      make([]string, 0, propcount),
	}

	for i := 0; i < propcount; i++ {
		name := file.readWZObjectUOL(parent.GetPath(), offset)
		if file.Debug {
			parent.debug(file, "Prop ", i, " has name ", name)
		}
		variant := NewWZVariant(name, parent)
		variant.Parse(file, offset)
		result.Properties[name] = variant
		result.Order = append(result.Order, name) // Track insertion order
	}

	return result
}
