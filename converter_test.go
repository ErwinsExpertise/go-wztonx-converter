package main

import (
	"testing"
)

func TestNodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		nodeType uint16
		expected string
	}{
		{"None", NodeTypeNone, "None"},
		{"Int64", NodeTypeInt64, "Int64"},
		{"Double", NodeTypeDouble, "Double"},
		{"String", NodeTypeString, "String"},
		{"Point", NodeTypePOINT, "Point"},
		{"Bitmap", NodeTypeBitmap, "Bitmap"},
		{"Audio", NodeTypeAudio, "Audio"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.nodeType > 6 {
				t.Errorf("Invalid node type: %d", tt.nodeType)
			}
		})
	}
}

func TestStringDeduplication(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", false, false)

	// Add the same string multiple times
	id1 := converter.addString("test")
	id2 := converter.addString("test")
	id3 := converter.addString("different")
	id4 := converter.addString("test")

	if id1 != id2 || id1 != id4 {
		t.Errorf("String deduplication failed: id1=%d, id2=%d, id4=%d", id1, id2, id4)
	}

	if id1 == id3 {
		t.Errorf("Different strings should have different IDs: id1=%d, id3=%d", id1, id3)
	}

	// Should have "test" and "different" = 2 strings
	if len(converter.strings) != 2 {
		t.Errorf("Expected 2 strings, got %d", len(converter.strings))
	}
}

func TestNodeFlattening(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", false, false)

	// Create a simple node tree
	root := &Node{
		Name:     "root",
		Type:     NodeTypeNone,
		Children: []*Node{},
	}

	child1 := &Node{
		Name:     "child1",
		Type:     NodeTypeInt64,
		Data:     int64(42),
		Children: []*Node{},
	}

	child2 := &Node{
		Name:     "child2",
		Type:     NodeTypeString,
		Data:     "hello",
		Children: []*Node{},
	}

	root.Children = append(root.Children, child1, child2)

	// Flatten the tree
	converter.flattenNodes(root)

	// Should have 3 nodes: root, child1, child2
	if len(converter.nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(converter.nodes))
	}

	// Check order is preserved (not sorted)
	if converter.nodes[0].Name != "root" {
		t.Errorf("Expected first node to be root, got %s", converter.nodes[0].Name)
	}
	if converter.nodes[1].Name != "child1" {
		t.Errorf("Expected second node to be child1, got %s", converter.nodes[1].Name)
	}
	if converter.nodes[2].Name != "child2" {
		t.Errorf("Expected third node to be child2, got %s", converter.nodes[2].Name)
	}
}

func TestColorTables(t *testing.T) {
	// Test table4
	if table4[0] != 0x00 || table4[15] != 0xFF {
		t.Error("table4 values incorrect")
	}

	// Test table5
	if table5[0] != 0x00 || table5[31] != 0xFF {
		t.Error("table5 values incorrect")
	}

	// Test table6
	if table6[0] != 0x00 || table6[63] != 0xFF {
		t.Error("table6 values incorrect")
	}
}

func TestRGB565Conversion(t *testing.T) {
	// Test converting a simple RGB565 image (1x1 pixel)
	data := []byte{0xFF, 0xFF} // White pixel in RGB565
	output, err := convertRGB565(data, 1, 1)

	if err != nil {
		t.Errorf("RGB565 conversion failed: %v", err)
	}

	if len(output) != 4 {
		t.Errorf("Expected 4 bytes (RGBA), got %d", len(output))
	}

	// Check that alpha is 255 (fully opaque)
	if output[3] != 255 {
		t.Errorf("Expected alpha to be 255, got %d", output[3])
	}
}

func TestARGB8888Conversion(t *testing.T) {
	// Test converting ARGB8888 (BGRA in WZ) to RGBA
	data := []byte{0xFF, 0x00, 0x00, 0x80} // Blue pixel with alpha
	output, err := convertARGB8888(data, 1, 1)

	if err != nil {
		t.Errorf("ARGB8888 conversion failed: %v", err)
	}

	if len(output) != 4 {
		t.Errorf("Expected 4 bytes (RGBA), got %d", len(output))
	}

	// Check color channel swap (BGRA -> RGBA)
	if output[0] != 0x00 || output[1] != 0x00 || output[2] != 0xFF || output[3] != 0x80 {
		t.Errorf("Color channels not swapped correctly: R=%d, G=%d, B=%d, A=%d",
			output[0], output[1], output[2], output[3])
	}
}
