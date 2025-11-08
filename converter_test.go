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

func TestNodeFlatteningWithNesting(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", false, false)

	// Create a more complex tree structure:
	// root
	//   ├─ child1
	//   │   └─ grandchild1
	//   └─ child2
	//       ├─ grandchild2
	//       └─ grandchild3

	grandchild1 := &Node{Name: "grandchild1", Type: NodeTypeInt64, Data: int64(1), Children: []*Node{}}
	grandchild2 := &Node{Name: "grandchild2", Type: NodeTypeInt64, Data: int64(2), Children: []*Node{}}
	grandchild3 := &Node{Name: "grandchild3", Type: NodeTypeInt64, Data: int64(3), Children: []*Node{}}

	child1 := &Node{
		Name:     "child1",
		Type:     NodeTypeNone,
		Children: []*Node{grandchild1},
	}

	child2 := &Node{
		Name:     "child2",
		Type:     NodeTypeNone,
		Children: []*Node{grandchild2, grandchild3},
	}

	root := &Node{
		Name:     "root",
		Type:     NodeTypeNone,
		Children: []*Node{child1, child2},
	}

	// Flatten the tree
	converter.flattenNodes(root)

	// Expected order with breadth-first:
	// 0: root
	// 1: child1
	// 2: child2
	// 3: grandchild1
	// 4: grandchild2
	// 5: grandchild3

	if len(converter.nodes) != 6 {
		t.Fatalf("Expected 6 nodes, got %d", len(converter.nodes))
	}

	expectedOrder := []string{"root", "child1", "child2", "grandchild1", "grandchild2", "grandchild3"}
	for i, expected := range expectedOrder {
		if converter.nodes[i].Name != expected {
			t.Errorf("Node at index %d: expected %s, got %s", i, expected, converter.nodes[i].Name)
		}
	}

	// Verify root's children are at indices 1 and 2 (contiguous)
	// Find root's first child index
	var rootFirstChild uint32
	for i, n := range converter.nodes {
		if n == root.Children[0] {
			rootFirstChild = uint32(i)
			break
		}
	}
	if rootFirstChild != 1 {
		t.Errorf("Root's first child should be at index 1, got %d", rootFirstChild)
	}
	// Second child should be at index 2 (rootFirstChild + 1)
	if converter.nodes[2] != root.Children[1] {
		t.Errorf("Root's second child should be at index 2")
	}

	// Verify child2's children are at indices 4 and 5 (contiguous)
	var child2FirstChild uint32
	for i, n := range converter.nodes {
		if n == child2.Children[0] {
			child2FirstChild = uint32(i)
			break
		}
	}
	if child2FirstChild != 4 {
		t.Errorf("Child2's first child should be at index 4, got %d", child2FirstChild)
	}
	if converter.nodes[5] != child2.Children[1] {
		t.Errorf("Child2's second child should be at index 5")
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

func TestScaleImage(t *testing.T) {
	// Test scaling a 2x2 image by 2x
	data := []byte{
		// Pixel 0,0: Red
		255, 0, 0, 255,
		// Pixel 1,0: Green
		0, 255, 0, 255,
		// Pixel 0,1: Blue
		0, 0, 255, 255,
		// Pixel 1,1: White
		255, 255, 255, 255,
	}

	scaled := scaleImage(data, 2, 2, 2)

	// Should now be 4x4 = 16 pixels = 64 bytes
	expectedSize := 4 * 4 * 4
	if len(scaled) != expectedSize {
		t.Errorf("Expected %d bytes for scaled image, got %d", expectedSize, len(scaled))
	}

	// Check that first pixel is still red (top-left corner)
	if scaled[0] != 255 || scaled[1] != 0 || scaled[2] != 0 || scaled[3] != 255 {
		t.Errorf("First pixel not red: R=%d, G=%d, B=%d, A=%d",
			scaled[0], scaled[1], scaled[2], scaled[3])
	}
}

func TestScaleImageNoScale(t *testing.T) {
	// Test that scale factor of 1 returns original data
	data := []byte{255, 0, 0, 255}
	scaled := scaleImage(data, 1, 1, 1)

	if len(scaled) != len(data) {
		t.Errorf("Scale factor 1 should not change size")
	}

	for i := range data {
		if scaled[i] != data[i] {
			t.Errorf("Scale factor 1 should return identical data")
			break
		}
	}
}

func TestParallelBitmapCompression(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", true, false)

	// Create test bitmap data
	testData := make([]byte, 1000)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Add multiple bitmaps
	for i := 0; i < 10; i++ {
		bitmap := BitmapData{
			Width:  10,
			Height: 10,
			Data:   testData,
		}
		converter.bitmaps = append(converter.bitmaps, bitmap)
	}

	// Compress in parallel
	err := converter.compressBitmapsParallel()
	if err != nil {
		t.Errorf("Parallel bitmap compression failed: %v", err)
	}

	// Verify all bitmaps were compressed
	for i, bitmap := range converter.bitmaps {
		if len(bitmap.CompressedData) == 0 {
			t.Errorf("Bitmap %d was not compressed", i)
		}
	}
}

func TestParallelCompressionWithEmptyBitmaps(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", true, false)

	// Add bitmaps with no data
	for i := 0; i < 5; i++ {
		bitmap := BitmapData{
			Width:  10,
			Height: 10,
			Data:   []byte{},
		}
		converter.bitmaps = append(converter.bitmaps, bitmap)
	}

	// Should not fail with empty bitmaps
	err := converter.compressBitmapsParallel()
	if err != nil {
		t.Errorf("Parallel compression should handle empty bitmaps: %v", err)
	}
}

func TestParallelCompressionWithAlreadyCompressed(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", true, false)

	// Add already compressed bitmaps
	for i := 0; i < 5; i++ {
		bitmap := BitmapData{
			Width:          10,
			Height:         10,
			Data:           []byte{1, 2, 3},
			CompressedData: []byte{4, 5, 6}, // Already compressed
		}
		converter.bitmaps = append(converter.bitmaps, bitmap)
	}

	// Should skip already compressed bitmaps
	err := converter.compressBitmapsParallel()
	if err != nil {
		t.Errorf("Parallel compression failed: %v", err)
	}

	// Verify compressed data was not changed
	for i, bitmap := range converter.bitmaps {
		if len(bitmap.CompressedData) != 3 || bitmap.CompressedData[0] != 4 {
			t.Errorf("Bitmap %d compressed data was modified", i)
		}
	}
}
