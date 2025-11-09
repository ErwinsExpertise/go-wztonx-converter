package main

import (
	"bytes"

	"github.com/pierrec/lz4/v4"
)

// compressLZ4 compresses data using LZ4
func compressLZ4(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lz4.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// compressLZ4HC compresses data using LZ4 High Compression
func compressLZ4HC(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lz4.NewWriter(&buf)
	// Set high compression level
	if err := writer.Apply(lz4.CompressionLevelOption(lz4.Level9)); err != nil {
		return nil, err
	}

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Compress data based on HC flag
func (c *Converter) compressData(data []byte) ([]byte, error) {
	if c.hc {
		return compressLZ4HC(data)
	}
	return compressLZ4(data)
}
