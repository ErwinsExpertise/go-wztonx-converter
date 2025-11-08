package main

import (
	"fmt"

	wz "github.com/diamondo25/go-wz"
)

// parseWZFile reads and parses the WZ file using the go-wz library
func (c *Converter) parseWZFile() error {
	wzFile, err := wz.NewFile(c.wzFilename)
	if err != nil {
		return fmt.Errorf("opening WZ file: %w", err)
	}
	defer wzFile.Close()

	wzFile.Parse()
	wzFile.WaitUntilLoaded()

	// Add empty string at index 0
	c.addString("")

	// Create root node
	root := &Node{
		Name:     "",
		Children: []*Node{},
		Type:     NodeTypeNone,
	}

	// Parse the WZ structure
	if wzFile.Root != nil {
		c.traverseWZDirectory(wzFile.Root, root)
	}

	// Flatten nodes into list (preserving order, NOT sorting)
	c.flattenNodes(root)

	return nil
}

// traverseWZDirectory recursively traverses WZ directories
func (c *Converter) traverseWZDirectory(wzDir *wz.WZDirectory, parentNode *Node) {
	// Process subdirectories
	for name, dir := range wzDir.Directories {
		childNode := &Node{
			Name:     name,
			Children: []*Node{},
			Type:     NodeTypeNone,
		}
		parentNode.Children = append(parentNode.Children, childNode)
		c.traverseWZDirectory(dir, childNode)
	}

	// Process images
	for name, img := range wzDir.Images {
		childNode := &Node{
			Name:     name,
			Children: []*Node{},
			Type:     NodeTypeNone,
		}
		parentNode.Children = append(parentNode.Children, childNode)
		c.traverseWZImage(img, childNode)
	}
}

// traverseWZImage processes a WZ image
func (c *Converter) traverseWZImage(wzImg *wz.WZImage, parentNode *Node) {
	wzImg.StartParse()

	for name, prop := range wzImg.Properties {
		c.traverseWZVariant(name, prop, parentNode)
	}
}

// traverseWZVariant processes a WZ variant
func (c *Converter) traverseWZVariant(name string, variant *wz.WZVariant, parentNode *Node) {
	node := &Node{
		Name:     name,
		Children: []*Node{},
	}

	switch variant.Type {
	case 0: // None
		node.Type = NodeTypeNone
		node.Data = nil

	case 2, 11: // int16
		node.Type = NodeTypeInt64
		if val, ok := variant.Value.(int16); ok {
			node.Data = int64(val)
		}

	case 3, 19: // int32
		node.Type = NodeTypeInt64
		if val, ok := variant.Value.(int32); ok {
			node.Data = int64(val)
		}

	case 20: // int64
		node.Type = NodeTypeInt64
		if val, ok := variant.Value.(int64); ok {
			node.Data = val
		}

	case 4: // float32
		node.Type = NodeTypeDouble
		if val, ok := variant.Value.(float32); ok {
			node.Data = float64(val)
		}

	case 5: // float64
		node.Type = NodeTypeDouble
		if val, ok := variant.Value.(float64); ok {
			node.Data = val
		}

	case 8: // String
		node.Type = NodeTypeString
		if val, ok := variant.Value.(string); ok {
			node.Data = val
		}

	case 9: // Sub object
		c.traverseWZObject(variant.Value, node)

	default:
		node.Type = NodeTypeNone
		node.Data = nil
	}

	parentNode.Children = append(parentNode.Children, node)
}

// traverseWZObject processes a WZ object (Canvas, Vector, Sound, etc.)
func (c *Converter) traverseWZObject(obj interface{}, parentNode *Node) {
	switch v := obj.(type) {
	case *wz.WZCanvas:
		c.traverseWZCanvas(v, parentNode)

	case *wz.WZVector:
		parentNode.Type = NodeTypePOINT
		parentNode.Data = [2]int32{v.X, v.Y}

	case *wz.WZSoundDX8:
		if c.client {
			c.traverseWZSound(v, parentNode)
		} else {
			parentNode.Type = NodeTypeNone
		}

	case wz.WZProperty:
		parentNode.Type = NodeTypeNone
		for name, prop := range v {
			c.traverseWZVariant(name, prop, parentNode)
		}

	case *wz.WZUOL:
		// Handle UOL (link) - for now, treat as None
		parentNode.Type = NodeTypeNone

	default:
		parentNode.Type = NodeTypeNone
	}
}

// traverseWZCanvas processes a Canvas (bitmap image)
func (c *Converter) traverseWZCanvas(canvas *wz.WZCanvas, parentNode *Node) {
	// Process canvas properties first
	if canvas.Properties != nil {
		for name, prop := range canvas.Properties {
			c.traverseWZVariant(name, prop, parentNode)
		}
	}

	// If in client mode, handle bitmap data
	if c.client && canvas.Width > 0 && canvas.Height > 0 {
		bitmapID := uint32(len(c.bitmaps))

		bitmap := BitmapData{
			Width:  uint16(canvas.Width),
			Height: uint16(canvas.Height),
			Data:   c.extractCanvasData(canvas),
		}
		c.bitmaps = append(c.bitmaps, bitmap)

		parentNode.Type = NodeTypeBitmap
		parentNode.Data = bitmapID
	} else {
		parentNode.Type = NodeTypeNone
	}
}

// extractCanvasData extracts and decompresses canvas pixel data
func (c *Converter) extractCanvasData(canvas *wz.WZCanvas) []byte {
	// Get the raw canvas data using reflection
	return getCanvasDataViaReflection(canvas)
}

// traverseWZSound processes a Sound object
func (c *Converter) traverseWZSound(sound *wz.WZSoundDX8, parentNode *Node) {
	audioID := uint32(len(c.audio))

	// Use reflection to access unexported soundData field
	soundData := getSoundDataViaReflection(sound)

	audio := AudioData{
		Length: uint32(len(soundData)),
		Data:   soundData,
	}
	c.audio = append(c.audio, audio)

	parentNode.Type = NodeTypeAudio
	parentNode.Data = audioID
}


