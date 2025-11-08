# go-wztonx-converter

A Go implementation of the WZ to NX file converter, based on [NoLifeStory's wztonx](https://github.com/NoLifeDev/NoLifeStory/blob/master/src/wztonx/wztonx.cpp).

## Overview

This tool converts MapleStory WZ files into the more efficient NX format. It uses the [go-wz](https://github.com/diamondo25/go-wz) library for reading WZ files.

## Key Differences from the C++ Version

- **Does NOT sort nodes** - Node order is preserved as-is from the WZ file
- Written in Go for better cross-platform support and memory safety
- Uses LZ4 compression for bitmaps and audio data

## Installation

```bash
go get github.com/ErwinsExpertise/go-wztonx-converter
```

Or build from source:

```bash
git clone https://github.com/ErwinsExpertise/go-wztonx-converter
cd go-wztonx-converter
go build
```

## Usage

```bash
# Convert a single WZ file
./go-wztonx-converter file.wz

# Convert with client mode (includes audio and bitmaps)
./go-wztonx-converter --client file.wz
./go-wztonx-converter -c file.wz

# Convert with server mode (no audio/bitmaps)
./go-wztonx-converter --server file.wz
./go-wztonx-converter -s file.wz

# Use high compression LZ4
./go-wztonx-converter --lz4hc file.wz
./go-wztonx-converter -h file.wz

# Convert entire directory
./go-wztonx-converter --client /path/to/wz/files/

# Combine options
./go-wztonx-converter -c -h file.wz
```

## Command Line Options

- `--client`, `-c`: Client mode - processes audio and bitmap data
- `--server`, `-s`: Server mode - skips audio and bitmap data
- `--lz4hc`, `-h`: Use LZ4 high compression (slower but smaller files)

## NX File Format

The NX file format consists of:

- **Header** (52 bytes): Contains magic number "PKG4" and offsets to various sections
- **Nodes**: Tree structure with different node types (int64, double, string, point, bitmap, audio)
- **String Table**: Deduplicated strings referenced by nodes
- **Bitmap Data**: LZ4-compressed image data (client mode only)
- **Audio Data**: Audio file data (client mode only)

## Node Types

- Type 0: None/Empty
- Type 1: Int64
- Type 2: Double (float64)
- Type 3: String
- Type 4: Point/Vector (x, y coordinates)
- Type 5: Bitmap (image data)
- Type 6: Audio (sound data)

## Technical Details

### Node Ordering

**Important**: Unlike the C++ version, this implementation **does NOT sort nodes**. Nodes are kept in their original order from the WZ file. This was a specific requirement to preserve the exact structure of the source data.

### Compression

- Bitmap data is compressed using LZ4 or LZ4HC
- Audio data is typically already compressed (MP3, etc.) and may not need additional compression
- LZ4HC provides better compression at the cost of slower encoding

### Memory-Mapped I/O

The go-wz library uses memory-mapped I/O for efficient WZ file reading.

## Dependencies

- [github.com/diamondo25/go-wz](https://github.com/diamondo25/go-wz) - WZ file parsing
- [github.com/pierrec/lz4/v4](https://github.com/pierrec/lz4) - LZ4 compression

## License

This project follows the same license as the original NoLifeStory project.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## Credits

- Original C++ implementation: [NoLifeStory by Peter Atashian](https://github.com/NoLifeDev/NoLifeStory)
- WZ parsing library: [go-wz by diamondo25](https://github.com/diamondo25/go-wz)
