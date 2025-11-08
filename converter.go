package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
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

	// Use buffered writer for better performance
	bufferedWriter := bufio.NewWriterSize(file, 1024*1024) // 1MB buffer
	defer bufferedWriter.Flush()

	return c.writeNXData(bufferedWriter)
}

// writeNXData writes the actual NX format data
func (c *Converter) writeNXData(w io.Writer) error {
	// Write header
	fmt.Print("  Writing header...")
	if err := c.writeHeader(w); err != nil {
		return err
	}
	fmt.Println("Done!")

	// Write nodes
	fmt.Printf("  Writing %d nodes...", len(c.nodes))
	if err := c.writeNodes(w); err != nil {
		return err
	}
	fmt.Println("Done!")

	// Write string table
	fmt.Printf("  Writing %d strings...", len(c.strings))
	if err := c.writeStrings(w); err != nil {
		return err
	}
	fmt.Println("Done!")

	// Write bitmaps and audio if in client mode
	if c.client {
		if len(c.bitmaps) > 0 {
			fmt.Printf("  Compressing %d bitmaps...", len(c.bitmaps))
			if err := c.compressBitmapsParallel(); err != nil {
				return err
			}
			fmt.Println("Done!")

			fmt.Print("  Writing bitmaps...")
			if err := c.writeBitmaps(w); err != nil {
				return err
			}
			fmt.Println("Done!")
		}

		if len(c.audio) > 0 {
			fmt.Printf("  Writing %d audio files...", len(c.audio))
			if err := c.writeAudio(w); err != nil {
				return err
			}
			fmt.Println("Done!")
		}
	}

	return nil
}

// writeHeader writes the NX file header
func (c *Converter) writeHeader(w io.Writer) error {
	// NX Header:
	// 4 bytes: magic "PKG4"
	// 4 bytes: node count
	// 8 bytes: node offset (52 bytes from start)
	// 4 bytes: string count
	// 8 bytes: string offset
	// 4 bytes: bitmap count
	// 8 bytes: bitmap offset
	// 4 bytes: audio count
	// 8 bytes: audio offset

	nodeCount := uint32(len(c.nodes))
	stringCount := uint32(len(c.strings))
	bitmapCount := uint32(len(c.bitmaps))
	audioCount := uint32(len(c.audio))

	// Calculate offsets
	nodeOffset := uint64(52) // Header size
	stringOffset := nodeOffset + uint64(nodeCount)*20
	bitmapOffset := stringOffset + c.calculateStringTableSize()
	audioOffset := bitmapOffset + c.calculateBitmapTableSize()

	// Write header
	if _, err := w.Write([]byte(NXMagic)); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, nodeCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, nodeOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, stringCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, stringOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitmapCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitmapOffset); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, audioCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, audioOffset); err != nil {
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
		bitmapID := node.Data.(uint32)
		if err = binary.Write(w, binary.LittleEndian, bitmapID); err != nil {
			return err
		}
		err = binary.Write(w, binary.LittleEndian, uint32(0)) // padding
	case NodeTypeAudio:
		audioID := node.Data.(uint32)
		if err = binary.Write(w, binary.LittleEndian, audioID); err != nil {
			return err
		}
		err = binary.Write(w, binary.LittleEndian, uint32(0)) // padding
	default:
		err = binary.Write(w, binary.LittleEndian, uint64(0))
	}
	return err
}

// writeStrings writes the string table
func (c *Converter) writeStrings(w io.Writer) error {
	for _, str := range c.strings {
		// String format:
		// 2 bytes: length
		// N bytes: UTF-8 string data
		length := uint16(len(str))
		if err := binary.Write(w, binary.LittleEndian, length); err != nil {
			return err
		}
		if _, err := w.Write([]byte(str)); err != nil {
			return err
		}
	}
	return nil
}

// writeBitmaps writes bitmap data
func (c *Converter) writeBitmaps(w io.Writer) error {
	// Calculate offsets
	currentOffset := uint64(0)
	for i := range c.bitmaps {
		c.bitmaps[i].Offset = currentOffset
		currentOffset += uint64(len(c.bitmaps[i].CompressedData)) + 4 // 4 bytes for size
	}

	// Write bitmap info table (width, height, offset)
	for _, bitmap := range c.bitmaps {
		if err := binary.Write(w, binary.LittleEndian, bitmap.Width); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, bitmap.Height); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, uint32(bitmap.Offset)); err != nil {
			return err
		}
	}

	// Write actual compressed bitmap data
	for _, bitmap := range c.bitmaps {
		// Write size of compressed data
		if err := binary.Write(w, binary.LittleEndian, uint32(len(bitmap.CompressedData))); err != nil {
			return err
		}
		// Write compressed data
		if _, err := w.Write(bitmap.CompressedData); err != nil {
			return err
		}
	}

	return nil
}

// writeAudio writes audio data
func (c *Converter) writeAudio(w io.Writer) error {
	// Calculate offsets first
	currentOffset := uint64(0)
	for i := range c.audio {
		c.audio[i].Offset = currentOffset

		// Audio data is typically already in a compressed format (MP3, etc.)
		// So we might not need to compress it again, but for consistency with C++ version,
		// we should still apply LZ4 if specified
		if len(c.audio[i].CompressedData) == 0 && len(c.audio[i].Data) > 0 {
			// For audio, we typically don't compress further as it's already compressed
			// But matching C++ behavior
			c.audio[i].CompressedData = c.audio[i].Data
		}

		currentOffset += uint64(len(c.audio[i].CompressedData))
	}

	// Write audio info table (length, offset)
	for _, audio := range c.audio {
		if err := binary.Write(w, binary.LittleEndian, audio.Length); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, uint32(audio.Offset)); err != nil {
			return err
		}
	}

	// Write actual audio data
	for _, audio := range c.audio {
		if _, err := w.Write(audio.CompressedData); err != nil {
			return err
		}
	}

	return nil
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

// calculateStringTableSize returns the size of the string table
func (c *Converter) calculateStringTableSize() uint64 {
	size := uint64(0)
	for _, str := range c.strings {
		size += 2 + uint64(len(str)) // 2 bytes for length + string data
	}
	return size
}

// calculateBitmapTableSize returns the size of the bitmap table
func (c *Converter) calculateBitmapTableSize() uint64 {
	// Each bitmap entry: 2 (width) + 2 (height) + 4 (offset) = 8 bytes
	return uint64(len(c.bitmaps)) * 8
}

// compressBitmapsParallel compresses all bitmap data in parallel
func (c *Converter) compressBitmapsParallel() error {
	if len(c.bitmaps) == 0 {
		return nil
	}

	// Create error channel and wait group
	errChan := make(chan error, len(c.bitmaps))
	var wg sync.WaitGroup

	// Limit concurrent goroutines to avoid overwhelming the system
	maxWorkers := 8
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
		for _, child := range node.Children {
			queue = append(queue, child)
		}
	}
}
