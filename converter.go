package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
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
	nodes      []*Node
	strings    []string
	stringMap  map[string]uint32
	bitmaps    []BitmapData
	audio      []AudioData
	bitmapData bytes.Buffer
	audioData  bytes.Buffer
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
	Width  uint16
	Height uint16
	Data   []byte
}

// AudioData stores audio information
type AudioData struct {
	Length uint32
	Data   []byte
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
	fmt.Print("Creating output.....")

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

	return c.writeNXData(file)
}

// writeNXData writes the actual NX format data
func (c *Converter) writeNXData(w io.Writer) error {
	// Write header
	if err := c.writeHeader(w); err != nil {
		return err
	}

	// Write nodes
	if err := c.writeNodes(w); err != nil {
		return err
	}

	// Write string table
	if err := c.writeStrings(w); err != nil {
		return err
	}

	// Write bitmaps and audio if in client mode
	if c.client {
		if err := c.writeBitmaps(w); err != nil {
			return err
		}
		if err := c.writeAudio(w); err != nil {
			return err
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

	binary.Write(w, binary.LittleEndian, nodeCount)
	binary.Write(w, binary.LittleEndian, nodeOffset)
	binary.Write(w, binary.LittleEndian, stringCount)
	binary.Write(w, binary.LittleEndian, stringOffset)
	binary.Write(w, binary.LittleEndian, bitmapCount)
	binary.Write(w, binary.LittleEndian, bitmapOffset)
	binary.Write(w, binary.LittleEndian, audioCount)
	binary.Write(w, binary.LittleEndian, audioOffset)

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

		binary.Write(w, binary.LittleEndian, nameID)
		binary.Write(w, binary.LittleEndian, firstChild)
		binary.Write(w, binary.LittleEndian, childCount)
		binary.Write(w, binary.LittleEndian, node.Type)

		// Write data based on type
		if err := c.writeNodeData(w, node); err != nil {
			return err
		}
	}

	return nil
}

// writeNodeData writes type-specific node data
func (c *Converter) writeNodeData(w io.Writer, node *Node) error {
	switch node.Type {
	case NodeTypeNone:
		binary.Write(w, binary.LittleEndian, uint64(0))
	case NodeTypeInt64:
		binary.Write(w, binary.LittleEndian, node.Data.(int64))
	case NodeTypeDouble:
		binary.Write(w, binary.LittleEndian, node.Data.(float64))
	case NodeTypeString:
		strID := c.getStringID(node.Data.(string))
		binary.Write(w, binary.LittleEndian, uint32(strID))
		binary.Write(w, binary.LittleEndian, uint32(0)) // padding
	case NodeTypePOINT:
		point := node.Data.([2]int32)
		binary.Write(w, binary.LittleEndian, point[0])
		binary.Write(w, binary.LittleEndian, point[1])
	case NodeTypeBitmap:
		bitmapID := node.Data.(uint32)
		binary.Write(w, binary.LittleEndian, bitmapID)
		binary.Write(w, binary.LittleEndian, uint32(0)) // padding
	case NodeTypeAudio:
		audioID := node.Data.(uint32)
		binary.Write(w, binary.LittleEndian, audioID)
		binary.Write(w, binary.LittleEndian, uint32(0)) // padding
	default:
		binary.Write(w, binary.LittleEndian, uint64(0))
	}
	return nil
}

// writeStrings writes the string table
func (c *Converter) writeStrings(w io.Writer) error {
	for _, str := range c.strings {
		// String format:
		// 2 bytes: length
		// N bytes: UTF-8 string data
		length := uint16(len(str))
		binary.Write(w, binary.LittleEndian, length)
		w.Write([]byte(str))
	}
	return nil
}

// writeBitmaps writes bitmap data
func (c *Converter) writeBitmaps(w io.Writer) error {
	// Write bitmap offset table
	for _, bitmap := range c.bitmaps {
		binary.Write(w, binary.LittleEndian, bitmap.Width)
		binary.Write(w, binary.LittleEndian, bitmap.Height)
		// Write offset to actual data (to be implemented with LZ4)
		binary.Write(w, binary.LittleEndian, uint32(0)) // placeholder
	}

	// Write actual bitmap data (LZ4 compressed)
	// TODO: Implement LZ4 compression for bitmap data
	w.Write(c.bitmapData.Bytes())
	return nil
}

// writeAudio writes audio data
func (c *Converter) writeAudio(w io.Writer) error {
	// Write audio offset table
	for _, audio := range c.audio {
		binary.Write(w, binary.LittleEndian, audio.Length)
		// Write offset to actual data
		binary.Write(w, binary.LittleEndian, uint32(0)) // placeholder
	}

	// Write actual audio data
	w.Write(c.audioData.Bytes())
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

// flattenNodes flattens the node tree into a list
// IMPORTANT: Preserves order, does NOT sort
func (c *Converter) flattenNodes(root *Node) {
	var flatten func(*Node)
	flatten = func(node *Node) {
		c.nodes = append(c.nodes, node)
		for _, child := range node.Children {
			flatten(child)
		}
	}
	flatten(root)
}

// Helper to convert string to null-terminated for compatibility
func nullTerminate(s string) string {
	if strings.HasSuffix(s, "\x00") {
		return s
	}
	return s + "\x00"
}
