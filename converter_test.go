package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
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

// Helper for testing - a seekable buffer
type seekableBuffer struct {
	buf    []byte
	pos    int64
	maxPos int64 // Track maximum position written
}

func newSeekableBuffer() *seekableBuffer {
	return &seekableBuffer{buf: make([]byte, 0, 1024*1024)}
}

func (s *seekableBuffer) Write(p []byte) (n int, err error) {
	// Extend buffer if needed
	minLen := int(s.pos) + len(p)
	if minLen > len(s.buf) {
		newBuf := make([]byte, minLen)
		copy(newBuf, s.buf)
		s.buf = newBuf
	}

	n = copy(s.buf[s.pos:], p)
	s.pos += int64(n)

	// Track max position
	if s.pos > s.maxPos {
		s.maxPos = s.pos
	}

	return n, nil
}

func (s *seekableBuffer) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case 0: // io.SeekStart
		abs = offset
	case 1: // io.SeekCurrent
		abs = s.pos + offset
	case 2: // io.SeekEnd
		abs = int64(len(s.buf)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if abs < 0 {
		return 0, fmt.Errorf("negative position")
	}

	s.pos = abs

	// Extend buffer if seeking beyond current length
	if int(s.pos) > len(s.buf) {
		newBuf := make([]byte, s.pos)
		copy(newBuf, s.buf)
		s.buf = newBuf
	}

	return abs, nil
}

func (s *seekableBuffer) Bytes() []byte {
	return s.buf[:s.maxPos]
}

// TestNXFileFormat validates the NX file format structure
func TestNXFileFormat(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", true, false)

	// Add test data
	converter.addString("")       // Empty string at index 0
	converter.addString("root")   // Index 1
	converter.addString("child1") // Index 2
	converter.addString("child2") // Index 3
	converter.addString("value")  // Index 4

	// Create test nodes
	root := &Node{
		Name:     "",
		Children: []*Node{},
		Type:     NodeTypeNone,
	}

	child1 := &Node{
		Name:     "child1",
		Children: []*Node{},
		Type:     NodeTypeInt64,
		Data:     int64(42),
	}

	child2 := &Node{
		Name:     "child2",
		Children: []*Node{},
		Type:     NodeTypeString,
		Data:     "value",
	}

	root.Children = append(root.Children, child1, child2)

	// Add a test bitmap
	bitmapData := make([]byte, 100)
	bitmap := BitmapData{
		Width:          10,
		Height:         10,
		Data:           bitmapData,
		CompressedData: []byte{1, 2, 3, 4}, // Fake compressed data
	}
	converter.bitmaps = append(converter.bitmaps, bitmap)

	bitmapNode := &Node{
		Name:     "bitmap",
		Children: []*Node{},
		Type:     NodeTypeBitmap,
		Data: BitmapNodeData{
			ID:     0,
			Width:  10,
			Height: 10,
		},
	}
	root.Children = append(root.Children, bitmapNode)

	// Add a test audio
	audioData := []byte{5, 6, 7, 8, 9}
	audio := AudioData{
		Length:         uint32(len(audioData)),
		Data:           audioData,
		CompressedData: audioData,
	}
	converter.audio = append(converter.audio, audio)

	audioNode := &Node{
		Name:     "audio",
		Children: []*Node{},
		Type:     NodeTypeAudio,
		Data: AudioNodeData{
			ID:     0,
			Length: uint32(len(audioData)),
		},
	}
	root.Children = append(root.Children, audioNode)

	// Flatten nodes
	converter.flattenNodes(root)

	// Write to buffer
	buf := newSeekableBuffer()
	err := converter.writeNXData(buf)
	if err != nil {
		t.Fatalf("Failed to write NX data: %v", err)
	}

	// Validate header
	reader := bytes.NewReader(buf.Bytes())

	// Check magic
	magic := make([]byte, 4)
	_, err = io.ReadFull(reader, magic)
	if err != nil {
		t.Fatalf("Failed to read magic: %v", err)
	}
	if string(magic) != "PKG4" {
		t.Errorf("Invalid magic: got %s, want PKG4", string(magic))
	}

	// Read header fields
	var nodeCount uint32
	var nodeOffset uint64
	var stringCount uint32
	var stringOffsetTableOffset uint64
	var bitmapCount uint32
	var bitmapOffsetTableOffset uint64
	var audioCount uint32
	var audioOffsetTableOffset uint64

	if err := binary.Read(reader, binary.LittleEndian, &nodeCount); err != nil {
		t.Fatalf("Failed to read node count: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &nodeOffset); err != nil {
		t.Fatalf("Failed to read node offset: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &stringCount); err != nil {
		t.Fatalf("Failed to read string count: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &stringOffsetTableOffset); err != nil {
		t.Fatalf("Failed to read string offset: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &bitmapCount); err != nil {
		t.Fatalf("Failed to read bitmap count: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &bitmapOffsetTableOffset); err != nil {
		t.Fatalf("Failed to read bitmap offset: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &audioCount); err != nil {
		t.Fatalf("Failed to read audio count: %v", err)
	}
	if err := binary.Read(reader, binary.LittleEndian, &audioOffsetTableOffset); err != nil {
		t.Fatalf("Failed to read audio offset: %v", err)
	}

	// Validate counts
	if nodeCount != uint32(len(converter.nodes)) {
		t.Errorf("Node count mismatch: got %d, want %d", nodeCount, len(converter.nodes))
	}
	if stringCount != uint32(len(converter.strings)) {
		t.Errorf("String count mismatch: got %d, want %d", stringCount, len(converter.strings))
	}
	if bitmapCount != uint32(len(converter.bitmaps)) {
		t.Errorf("Bitmap count mismatch: got %d, want %d", bitmapCount, len(converter.bitmaps))
	}
	if audioCount != uint32(len(converter.audio)) {
		t.Errorf("Audio count mismatch: got %d, want %d", audioCount, len(converter.audio))
	}

	// Validate node offset
	if nodeOffset != 52 {
		t.Errorf("Node offset should be 52 (header size), got %d", nodeOffset)
	}

	// Validate string offset table is after strings
	if stringOffsetTableOffset <= nodeOffset {
		t.Errorf("String offset table should be after nodes, got %d", stringOffsetTableOffset)
	}

	// Validate bitmap offset table is after bitmaps
	if bitmapOffsetTableOffset <= stringOffsetTableOffset {
		t.Errorf("Bitmap offset table should be after string offset table, got %d", bitmapOffsetTableOffset)
	}

	// Validate audio offset table is after audio
	if audioOffsetTableOffset <= bitmapOffsetTableOffset {
		t.Errorf("Audio offset table should be after bitmap offset table, got %d", audioOffsetTableOffset)
	}

	t.Logf("Header validation passed:")
	t.Logf("  Node count: %d at offset %d", nodeCount, nodeOffset)
	t.Logf("  String count: %d, offset table at %d", stringCount, stringOffsetTableOffset)
	t.Logf("  Bitmap count: %d, offset table at %d", bitmapCount, bitmapOffsetTableOffset)
	t.Logf("  Audio count: %d, offset table at %d", audioCount, audioOffsetTableOffset)
}

// TestNXFileFormatReading tests that we can read back what we write
func TestNXFileFormatReading(t *testing.T) {
	converter := NewConverter("test.wz", "test.nx", true, false)

	// Add test strings
	converter.addString("")
	converter.addString("testStr1")
	converter.addString("testStr2")

	// Create simple node tree
	root := &Node{
		Name:     "",
		Children: []*Node{},
		Type:     NodeTypeNone,
	}

	stringNode := &Node{
		Name:     "testStr1",
		Children: []*Node{},
		Type:     NodeTypeString,
		Data:     "testStr2",
	}
	root.Children = append(root.Children, stringNode)

	// Add bitmap
	bitmap := BitmapData{
		Width:          5,
		Height:         10,
		Data:           make([]byte, 200),
		CompressedData: []byte{1, 2, 3},
	}
	converter.bitmaps = append(converter.bitmaps, bitmap)

	bitmapNode := &Node{
		Name:     "bitmap",
		Children: []*Node{},
		Type:     NodeTypeBitmap,
		Data: BitmapNodeData{
			ID:     0,
			Width:  5,
			Height: 10,
		},
	}
	root.Children = append(root.Children, bitmapNode)

	// Add audio
	audioData := []byte{0xAA, 0xBB, 0xCC}
	audio := AudioData{
		Length:         3,
		Data:           audioData,
		CompressedData: audioData,
	}
	converter.audio = append(converter.audio, audio)

	audioNode := &Node{
		Name:     "audio",
		Children: []*Node{},
		Type:     NodeTypeAudio,
		Data: AudioNodeData{
			ID:     0,
			Length: 3,
		},
	}
	root.Children = append(root.Children, audioNode)

	converter.flattenNodes(root)

	// Write to buffer
	buf := newSeekableBuffer()
	err := converter.writeNXData(buf)
	if err != nil {
		t.Fatalf("Failed to write NX data: %v", err)
	}

	// Now read back like gonx does
	reader := bytes.NewReader(buf.Bytes())

	// Read header
	var header struct {
		Magic                   [4]byte
		NodeCount               uint32
		NodeBlockOffset         int64
		StringCount             uint32
		StringOffsetTableOffset int64
		BitmapCount             uint32
		BitmapOffsetTableOffset int64
		AudioCount              uint32
		AudioOffsetTableOffset  int64
	}

	err = binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		t.Fatalf("Failed to read header: %v", err)
	}

	// Validate magic
	// Validate magic
	if string(header.Magic[:]) != "PKG4" {
		t.Errorf("Invalid magic: %s", string(header.Magic[:]))
	}

	t.Logf("Header values:")
	t.Logf("  String count: %d, offset table offset: %d", header.StringCount, header.StringOffsetTableOffset)
	t.Logf("  Buffer size: %d", len(buf.Bytes()))
	t.Logf("  Converter strings: %d", len(converter.strings))
	if string(header.Magic[:]) != "PKG4" {
		t.Errorf("Invalid magic: %s", string(header.Magic[:]))
	}

	// Read string offset table
	_, err = reader.Seek(header.StringOffsetTableOffset, 0)
	if err != nil {
		t.Fatalf("Failed to seek to string offset table: %v", err)
	}

	stringOffsets := make([]int64, header.StringCount)
	err = binary.Read(reader, binary.LittleEndian, &stringOffsets)
	if err != nil {
		t.Fatalf("Failed to read string offsets: %v", err)
	}

	// Read strings using offset table
	for i, offset := range stringOffsets {
		_, err = reader.Seek(offset, 0)
		if err != nil {
			t.Fatalf("Failed to seek to string %d: %v", i, err)
		}

		var length uint16
		err = binary.Read(reader, binary.LittleEndian, &length)
		if err != nil {
			t.Fatalf("Failed to read string length: %v", err)
		}

		strBytes := make([]byte, length)
		_, err = reader.Read(strBytes)
		if err != nil {
			t.Fatalf("Failed to read string data: %v", err)
		}

		t.Logf("String %d: %q", i, string(strBytes))
	}

	// Read bitmap offset table
	_, err = reader.Seek(header.BitmapOffsetTableOffset, 0)
	if err != nil {
		t.Fatalf("Failed to seek to bitmap offset table: %v", err)
	}

	bitmapOffsets := make([]int64, header.BitmapCount)
	err = binary.Read(reader, binary.LittleEndian, &bitmapOffsets)
	if err != nil {
		t.Fatalf("Failed to read bitmap offsets: %v", err)
	}

	// Read bitmaps using offset table
	for i, offset := range bitmapOffsets {
		_, err = reader.Seek(offset, 0)
		if err != nil {
			t.Fatalf("Failed to seek to bitmap %d: %v", i, err)
		}

		var width, height uint16
		var size uint32
		if err := binary.Read(reader, binary.LittleEndian, &width); err != nil {
			t.Fatalf("Failed to read bitmap width: %v", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &height); err != nil {
			t.Fatalf("Failed to read bitmap height: %v", err)
		}
		if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
			t.Fatalf("Failed to read bitmap size: %v", err)
		}

		bitmapData := make([]byte, size)
		_, err = reader.Read(bitmapData)
		if err != nil {
			t.Fatalf("Failed to read bitmap data: %v", err)
		}

		t.Logf("Bitmap %d: %dx%d, %d bytes", i, width, height, size)

		// Validate bitmap data
		if width != 5 || height != 10 {
			t.Errorf("Bitmap dimensions mismatch: got %dx%d, want 5x10", width, height)
		}
	}

	// Read audio offset table
	_, err = reader.Seek(header.AudioOffsetTableOffset, 0)
	if err != nil {
		t.Fatalf("Failed to seek to audio offset table: %v", err)
	}

	audioOffsets := make([]int64, header.AudioCount)
	err = binary.Read(reader, binary.LittleEndian, &audioOffsets)
	if err != nil {
		t.Fatalf("Failed to read audio offsets: %v", err)
	}

	// Read audio using offset table
	// Note: Audio length comes from the node data, not the audio section
	for i, offset := range audioOffsets {
		_, err = reader.Seek(offset, 0)
		if err != nil {
			t.Fatalf("Failed to seek to audio %d: %v", i, err)
		}

		// For this test, we know the length is 3
		audioBytes := make([]byte, 3)
		_, err = reader.Read(audioBytes)
		if err != nil {
			t.Fatalf("Failed to read audio data: %v", err)
		}

		t.Logf("Audio %d: %d bytes, data=%v", i, len(audioBytes), audioBytes)

		// Validate audio data
		if !bytes.Equal(audioBytes, []byte{0xAA, 0xBB, 0xCC}) {
			t.Errorf("Audio data mismatch: got %v, want [0xAA, 0xBB, 0xCC]", audioBytes)
		}
	}

	t.Log("Successfully read back all data from NX file")
}

// BenchmarkWriteWithBuffering benchmarks writing with buffered I/O
func BenchmarkWriteWithBuffering(b *testing.B) {
	// Create a converter with test data
	converter := NewConverter("test.wz", "test.nx", true, false)

	// Add some test strings
	for i := 0; i < 1000; i++ {
		converter.addString(fmt.Sprintf("string_%d", i))
	}

	// Create a large node tree
	root := &Node{
		Name:     "",
		Children: []*Node{},
		Type:     NodeTypeNone,
	}

	for i := 0; i < 100; i++ {
		child := &Node{
			Name:     fmt.Sprintf("child_%d", i),
			Children: []*Node{},
			Type:     NodeTypeInt64,
			Data:     int64(i),
		}
		root.Children = append(root.Children, child)
	}

	// Add test bitmaps with realistic sizes
	for i := 0; i < 50; i++ {
		bitmapData := make([]byte, 1024*10) // 10KB each
		for j := range bitmapData {
			bitmapData[j] = byte(j % 256)
		}
		bitmap := BitmapData{
			Width:          100,
			Height:         100,
			Data:           bitmapData,
			CompressedData: bitmapData[:len(bitmapData)/2], // Simulate compression
		}
		converter.bitmaps = append(converter.bitmaps, bitmap)
	}

	// Add test audio
	for i := 0; i < 10; i++ {
		audioData := make([]byte, 1024*50) // 50KB each
		audio := AudioData{
			Length:         uint32(len(audioData)),
			Data:           audioData,
			CompressedData: audioData,
		}
		converter.audio = append(converter.audio, audio)
	}

	converter.flattenNodes(root)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Use a temporary file
		tmpFile := fmt.Sprintf("/tmp/bench_test_%d.nx", i)
		converter.nxFilename = tmpFile
		b.StartTimer()

		// Write the file
		if err := converter.writeNXFile(); err != nil {
			b.Fatalf("Failed to write NX file: %v", err)
		}

		b.StopTimer()
		// Clean up
		os.Remove(tmpFile)
		b.StartTimer()
	}
}

// BenchmarkBufferedSeekerWrite benchmarks the buffered seeker's write performance
func BenchmarkBufferedSeekerWrite(b *testing.B) {
	tmpFile := "/tmp/buffered_seeker_bench.dat"
	defer os.Remove(tmpFile)

	// Create test data
	data := make([]byte, 1024) // 1KB
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.Run("Buffered4MB", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			file, _ := os.Create(tmpFile)
			bs := newBufferedSeeker(file, 4*1024*1024)
			b.StartTimer()

			// Write data many times
			for j := 0; j < 1000; j++ {
				bs.Write(data)
			}
			bs.Flush()

			b.StopTimer()
			file.Close()
			os.Remove(tmpFile)
			b.StartTimer()
		}
	})

	b.Run("Unbuffered", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			file, _ := os.Create(tmpFile)
			b.StartTimer()

			// Write data many times (unbuffered)
			for j := 0; j < 1000; j++ {
				file.Write(data)
			}

			b.StopTimer()
			file.Close()
			os.Remove(tmpFile)
			b.StartTimer()
		}
	})
}
