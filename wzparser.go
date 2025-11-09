package main

import (
	"fmt"
	"sync"

	"github.com/ErwinsExpertise/go-wztonx-converter/wz"
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
	// Process subdirectories in order
	for _, name := range wzDir.DirectoryOrder {
		dir := wzDir.Directories[name]
		childNode := &Node{
			Name:     name,
			Children: []*Node{},
			Type:     NodeTypeNone,
		}
		parentNode.Children = append(parentNode.Children, childNode)
		c.traverseWZDirectory(dir, childNode)
	}

	// Process images in parallel for better performance
	// Since images are independent, we can parse them concurrently
	if len(wzDir.ImageOrder) > 0 {
		// Create a slice to hold child nodes in order
		imageNodes := make([]*Node, len(wzDir.ImageOrder))
		var wg sync.WaitGroup

		for i, name := range wzDir.ImageOrder {
			imageNodes[i] = &Node{
				Name:     name,
				Children: []*Node{},
				Type:     NodeTypeNone,
			}

			wg.Add(1)
			// Capture loop variables
			img := wzDir.Images[name]
			node := imageNodes[i]

			go func() {
				defer wg.Done()
				c.traverseWZImage(img, node)
			}()
		}

		// Wait for all images to be processed
		wg.Wait()

		// Append nodes in order after parallel processing
		for _, node := range imageNodes {
			parentNode.Children = append(parentNode.Children, node)
		}
	}
}

// traverseWZImage processes a WZ image
func (c *Converter) traverseWZImage(wzImg *wz.WZImage, parentNode *Node) {
	wzImg.StartParse()

	if wzImg.Properties != nil {
		for _, name := range wzImg.Properties.Order {
			prop := wzImg.Properties.Properties[name]
			c.traverseWZVariant(name, prop, parentNode)
		}
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

	case *wz.WZProperty:
		parentNode.Type = NodeTypeNone
		for _, name := range v.Order {
			prop := v.Properties[name]
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
		for _, name := range canvas.Properties.Order {
			prop := canvas.Properties.Properties[name]
			c.traverseWZVariant(name, prop, parentNode)
		}
	}

	// If in client mode, handle bitmap data
	if c.client && canvas.Width > 0 && canvas.Height > 0 {
		bitmapID := uint32(len(c.bitmaps))
		width := uint16(canvas.Width)
		height := uint16(canvas.Height)

		bitmap := BitmapData{
			Width:  width,
			Height: height,
			Data:   c.extractCanvasData(canvas),
		}
		c.bitmaps = append(c.bitmaps, bitmap)

		parentNode.Type = NodeTypeBitmap
		parentNode.Data = BitmapNodeData{
			ID:     bitmapID,
			Width:  width,
			Height: height,
		}
	} else {
		parentNode.Type = NodeTypeNone
	}
}

// extractCanvasData extracts and decompresses canvas pixel data
func (c *Converter) extractCanvasData(canvas *wz.WZCanvas) []byte {
	// Get the canvas data using exported Data field
	rawData := canvas.Data

	if len(rawData) == 0 {
		return nil
	}

	// Process the canvas data based on its format
	processedData, err := processCanvasData(canvas, rawData)
	if err != nil {
		// Log error but don't fail completely
		fmt.Printf("Warning: Error processing canvas data: %v\n", err)
		return nil
	}

	return processedData
}

// traverseWZSound processes a Sound object
func (c *Converter) traverseWZSound(sound *wz.WZSoundDX8, parentNode *Node) {
	audioID := uint32(len(c.audio))

	// Use exported SoundData field directly
	soundData := sound.SoundData
	length := uint32(len(soundData))

	audio := AudioData{
		Length: length,
		Data:   soundData,
	}
	c.audio = append(c.audio, audio)

	parentNode.Type = NodeTypeAudio
	parentNode.Data = AudioNodeData{
		ID:     audioID,
		Length: length,
	}
}
