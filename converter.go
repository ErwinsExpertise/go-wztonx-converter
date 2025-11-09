package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
)

// NX file format constants
const (
	NXMagic = "PKG4"
)

// Node types
const (
	NodeTypeNone   = 0
	NodeTypeInt64  = 1
	NodeTypeDouble = 2
	NodeTypeString = 3
	NodeTypePOINT  = 4 // Vector (x, y)
	NodeTypeBitmap = 5
	NodeTypeAudio  = 6
)

// Converter handles the conversion from WZ to NX format
type Converter struct {
	wzFilename string
	nxFilename string
	client     bool
	hc         bool

	// NX data structures
	nodes     []*Node
	strings   []string
	stringMap map[string]uint32
	bitmaps   []BitmapData
	audio     []AudioData
}

// Node represents a node in the NX file
type Node struct {
	Name     string
	Children []*Node
	Type     uint16
	Data     interface{}
}

// BitmapNodeData stores bitmap node information
type BitmapNodeData struct {
	ID     uint32
	Width  uint16
	Height uint16
}

// AudioNodeData stores audio node information
type AudioNodeData struct {
	ID     uint32
	Length uint32
}

// BitmapData stores bitmap information
type BitmapData struct {
	Width          uint16
	Height         uint16
	Data           []byte
	CompressedData []byte
	Offset         uint64
}

// AudioData stores audio information
type AudioData struct {
	Length         uint32
	Data           []byte
	CompressedData []byte
	Offset         uint64
}

// NewConverter creates a new converter instance
func NewConverter(wzFile, nxFile string, client, hc bool) *Converter {
	return &Converter{
		wzFilename: wzFile,
		nxFilename: nxFile,
		client:     client,
		hc:         hc,
		stringMap:  make(map[string]uint32),
	}
}

// Convert performs the WZ to NX conversion
func (c *Converter) Convert() error {
	fmt.Print("Parsing input.......")

	// Parse WZ file
	if err := c.parseWZFile(); err != nil {
		return fmt.Errorf("parsing WZ file: %w", err)
	}

	fmt.Println("Done!")
	fmt.Println("Creating output.....")

	// Write NX file
	if err := c.writeNXFile(); err != nil {
		return fmt.Errorf("writing NX file: %w", err)
	}

	fmt.Println("Done!")
	return nil
}

// parseWZFile is implemented in wzparser.go

// writeNXFile writes the NX format file
func (c *Converter) writeNXFile() error {
	file, err := os.Create(c.nxFilename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Pass file directly as it implements io.WriteSeeker
	// writeNXData uses seeking to update the header after writing all data
	return c.writeNXData(file)
}

// writeNXData writes the actual NX format data
func (c *Converter) writeNXData(w io.Writer) error {
	// We need to use a seekable writer to update the header later
	// Cast to io.WriteSeeker
	seeker, ok := w.(io.WriteSeeker)
	if !ok {
		return fmt.Errorf("writer must support seeking")
	}

	// Write placeholder header
	fmt.Print("  Writing header...")
	if err := c.writeHeader(w); err != nil {
		return err
	}
	fmt.Println("Done!")

	// Write nodes
	fmt.Printf("  Writing %d nodes...", len(c.nodes))
	nodeOffset := uint64(52) // Header size
	if err := c.writeNodes(w); err != nil {
		return err
	}
	fmt.Println("Done!")

	// Write string data and offset table
	fmt.Printf("  Writing %d strings...", len(c.strings))
	stringOffsetTableOffset, err := c.writeStrings(w)
	if err != nil {
		return err
	}
	fmt.Println("Done!")

	// Write bitmaps and audio if in client mode
	var bitmapOffsetTableOffset uint64
	var audioOffsetTableOffset uint64

	if c.client {
		if len(c.bitmaps) > 0 {
			fmt.Printf("  Compressing %d bitmaps...", len(c.bitmaps))
			if err := c.compressBitmapsParallel(); err != nil {
				return err
			}
			fmt.Println("Done!")

			fmt.Print("  Writing bitmaps...")
			bitmapOffsetTableOffset, err = c.writeBitmaps(w)
			if err != nil {
				return err
			}
			fmt.Println("Done!")
		}

		if len(c.audio) > 0 {
			fmt.Printf("  Writing %d audio files...", len(c.audio))
			audioOffsetTableOffset, err = c.writeAudio(w)
			if err != nil {
				return err
			}
			fmt.Println("Done!")
		}
	}

	// Update header with actual offsets
	fmt.Print("  Finalizing header...")
	if err := c.updateHeader(seeker, nodeOffset, stringOffsetTableOffset, bitmapOffsetTableOffset, audioOffsetTableOffset); err != nil {
		return err
	}
	fmt.Println("Done!")

	return nil
}

// writeHeader writes the NX file header (placeholder values initially)
func (c *Converter) writeHeader(w io.Writer) error {
	// NX Header:
	// 4 bytes: magic "PKG4"
	// 4 bytes: node count
	// 8 bytes: node offset (52 bytes from start)
	// 4 bytes: string count
	// 8 bytes: string offset table offset
	// 4 bytes: bitmap count
	// 8 bytes: bitmap offset table offset
	// 4 bytes: audio count
	// 8 bytes: audio offset table offset

	// Write magic
	if _, err := w.Write([]byte(NXMagic)); err != nil {
		return err
	}

	// Write placeholder values (will be updated later)
	if err := binary.Write(w, binary.LittleEndian, uint32(0)); err != nil { // node count
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint64(0)); err != nil { // node offset
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(0)); err != nil { // string count
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint64(0)); err != nil { // string offset table offset
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(0)); err != nil { // bitmap count
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint64(0)); err != nil { // bitmap offset table offset
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(0)); err != nil { // audio count
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint64(0)); err != nil { // audio offset table offset
		return err
	}

	return nil
}

// writeNodes writes all nodes to the file
// IMPORTANT: Does NOT sort nodes - preserves original order
func (c *Converter) writeNodes(w io.Writer) error {
	// Node structure (20 bytes):
	// 4 bytes: name string ID
	// 4 bytes: first child index
	// 2 bytes: child count
	// 2 bytes: type
	// 8 bytes: data (type-dependent)

	for _, node := range c.nodes {
		nameID := c.getStringID(node.Name)

		// Calculate child info
		var firstChild uint32 = 0
		var childCount uint16 = 0
		if len(node.Children) > 0 {
			// Find index of first child
			for i, n := range c.nodes {
				if n == node.Children[0] {
					firstChild = uint32(i)
					break
				}
			}
			childCount = uint16(len(node.Children))
		}

		if err := binary.Write(w, binary.LittleEndian, nameID); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, firstChild); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, childCount); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, node.Type); err != nil {
			return err
		}

		// Write data based on type
		if err := c.writeNodeData(w, node); err != nil {
			return err
		}
	}

	return nil
}

// writeNodeData writes type-specific node data
func (c *Converter) writeNodeData(w io.Writer, node *Node) error {
	var err error
	switch node.Type {
	case NodeTypeNone:
		err = binary.Write(w, binary.LittleEndian, uint64(0))
	case NodeTypeInt64:
		err = binary.Write(w, binary.LittleEndian, node.Data.(int64))
	case NodeTypeDouble:
		err = binary.Write(w, binary.LittleEndian, node.Data.(float64))
	case NodeTypeString:
		strID := c.getStringID(node.Data.(string))
		if err = binary.Write(w, binary.LittleEndian, uint32(strID)); err != nil {
			return err
		}
		err = binary.Write(w, binary.LittleEndian, uint32(0)) // padding
	case NodeTypePOINT:
		point := node.Data.([2]int32)
		if err = binary.Write(w, binary.LittleEndian, point[0]); err != nil {
			return err
		}
		err = binary.Write(w, binary.LittleEndian, point[1])
	case NodeTypeBitmap:
		bitmapData := node.Data.(BitmapNodeData)
		if err = binary.Write(w, binary.LittleEndian, bitmapData.ID); err != nil {
			return err
		}
		if err = binary.Write(w, binary.LittleEndian, bitmapData.Width); err != nil {
			return err
		}
		err = binary.Write(w, binary.LittleEndian, bitmapData.Height)
	case NodeTypeAudio:
		audioData := node.Data.(AudioNodeData)
		if err = binary.Write(w, binary.LittleEndian, audioData.ID); err != nil {
			return err
		}
		err = binary.Write(w, binary.LittleEndian, audioData.Length)
	default:
		err = binary.Write(w, binary.LittleEndian, uint64(0))
	}
	return err
}

// updateHeader updates the header with final offset values
func (c *Converter) updateHeader(w io.WriteSeeker, nodeOffset, stringOffsetTableOffset, bitmapOffsetTableOffset, audioOffsetTableOffset uint64) error {
	// Seek to start of file (after magic)
	if _, err := w.Seek(4, io.SeekStart); err != nil {
		return err
	}

	nodeCount := uint32(len(c.nodes))
	stringCount := uint32(len(c.strings))
	bitmapCount := uint32(len(c.bitmaps))
	audioCount := uint32(len(c.audio))

	// Write actual values
	if err := binary.Write(w, binary.LittleEndian, nodeCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, nodeOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, stringCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, stringOffsetTableOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitmapCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitmapOffsetTableOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, audioCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, audioOffsetTableOffset); err != nil {
		return err
	}

	return nil
}

// writeStrings writes the string data and offset table
// Returns the offset to the offset table
func (c *Converter) writeStrings(w io.Writer) (uint64, error) {
	seeker, ok := w.(io.WriteSeeker)
	if !ok {
		return 0, fmt.Errorf("writer must support seeking")
	}

	// Store offsets for each string
	stringOffsets := make([]uint64, len(c.strings))

	// Write string data first
	for i, str := range c.strings {
		// Record current position as the string offset
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		stringOffsets[i] = uint64(pos)

		// String format:
		// 2 bytes: length
		// N bytes: UTF-8 string data
		length := uint16(len(str))
		if err := binary.Write(w, binary.LittleEndian, length); err != nil {
			return 0, err
		}
		if _, err := w.Write([]byte(str)); err != nil {
			return 0, err
		}
	}

	// Get position for offset table
	pos, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	stringOffsetTableOffset := uint64(pos)

	// Write offset table
	for _, offset := range stringOffsets {
		if err := binary.Write(w, binary.LittleEndian, offset); err != nil {
			return 0, err
		}
	}

	return stringOffsetTableOffset, nil
}

// writeBitmaps writes bitmap data and offset table
// Returns the offset to the offset table
func (c *Converter) writeBitmaps(w io.Writer) (uint64, error) {
	seeker, ok := w.(io.WriteSeeker)
	if !ok {
		return 0, fmt.Errorf("writer must support seeking")
	}

	// Store offsets for each bitmap
	bitmapOffsets := make([]uint64, len(c.bitmaps))

	// Write bitmap data first
	for i, bitmap := range c.bitmaps {
		// Record current position as the bitmap offset
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		bitmapOffsets[i] = uint64(pos)

		// Bitmap format:
		// 2 bytes: width
		// 2 bytes: height
		// 4 bytes: compressed data size
		// N bytes: compressed data
		if err := binary.Write(w, binary.LittleEndian, bitmap.Width); err != nil {
			return 0, err
		}
		if err := binary.Write(w, binary.LittleEndian, bitmap.Height); err != nil {
			return 0, err
		}
		// Write size of compressed data
		if err := binary.Write(w, binary.LittleEndian, uint32(len(bitmap.CompressedData))); err != nil {
			return 0, err
		}
		// Write compressed data
		if _, err := w.Write(bitmap.CompressedData); err != nil {
			return 0, err
		}
	}

	// Get position for offset table
	pos, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	bitmapOffsetTableOffset := uint64(pos)

	// Write offset table
	for _, offset := range bitmapOffsets {
		if err := binary.Write(w, binary.LittleEndian, offset); err != nil {
			return 0, err
		}
	}

	return bitmapOffsetTableOffset, nil
}

// writeAudio writes audio data and offset table
// Returns the offset to the offset table
func (c *Converter) writeAudio(w io.Writer) (uint64, error) {
	seeker, ok := w.(io.WriteSeeker)
	if !ok {
		return 0, fmt.Errorf("writer must support seeking")
	}

	// Store offsets for each audio
	audioOffsets := make([]uint64, len(c.audio))

	// Write audio data first
	for i, audio := range c.audio {
		// Record current position as the audio offset
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		audioOffsets[i] = uint64(pos)

		// Ensure we have compressed data
		if len(audio.CompressedData) == 0 && len(audio.Data) > 0 {
			// For audio, we typically don't compress further as it's already compressed
			// But matching C++ behavior
			c.audio[i].CompressedData = audio.Data
		}

		// Write audio data directly (no length prefix in the data section)
		if _, err := w.Write(c.audio[i].CompressedData); err != nil {
			return 0, err
		}
	}

	// Get position for offset table
	pos, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	audioOffsetTableOffset := uint64(pos)

	// Write offset table
	for _, offset := range audioOffsets {
		if err := binary.Write(w, binary.LittleEndian, offset); err != nil {
			return 0, err
		}
	}

	return audioOffsetTableOffset, nil
}

// addString adds a string to the string table and returns its ID
func (c *Converter) addString(str string) uint32 {
	if id, exists := c.stringMap[str]; exists {
		return id
	}
	id := uint32(len(c.strings))
	c.strings = append(c.strings, str)
	c.stringMap[str] = id
	return id
}

// getStringID returns the ID for a string
func (c *Converter) getStringID(str string) uint32 {
	if id, exists := c.stringMap[str]; exists {
		return id
	}
	return c.addString(str)
}

// compressBitmapsParallel compresses all bitmap data in parallel
func (c *Converter) compressBitmapsParallel() error {
	if len(c.bitmaps) == 0 {
		return nil
	}

	// Create error channel and wait group
	errChan := make(chan error, len(c.bitmaps))
	var wg sync.WaitGroup

	// Use more workers for better CPU utilization
	// Use 2x CPU count or at least 16 workers for good parallelism
	maxWorkers := runtime.NumCPU() * 2
	if maxWorkers < 16 {
		maxWorkers = 16
	}
	semaphore := make(chan struct{}, maxWorkers)

	for i := range c.bitmaps {
		// Skip if already compressed or no data
		if len(c.bitmaps[i].CompressedData) > 0 || len(c.bitmaps[i].Data) == 0 {
			continue
		}

		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Compress the bitmap data
			compressed, err := c.compressData(c.bitmaps[index].Data)
			if err != nil {
				errChan <- fmt.Errorf("compressing bitmap %d: %w", index, err)
				return
			}
			c.bitmaps[index].CompressedData = compressed
		}(i)
	}

	// Wait for all compressions to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// flattenNodes flattens the node tree into a list
// IMPORTANT: Ensures each parent's children are stored contiguously in the array,
// as required by the NX format (children at indices [firstChild, firstChild+count-1])
func (c *Converter) flattenNodes(root *Node) {
	var queue []*Node
	queue = append(queue, root)

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		c.nodes = append(c.nodes, node)

		// Add all children to the queue so they get added contiguously
		queue = append(queue, node.Children...)
	}
}
