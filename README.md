# go-wztonx-converter

A Go implementation of the WZ to NX file converter, based on [NoLifeStory's wztonx](https://github.com/NoLifeDev/NoLifeStory/blob/master/src/wztonx/wztonx.cpp).

## Overview

This tool converts MapleStory WZ files into the more efficient NX format. It includes a forked version of the [go-wz](https://github.com/diamondo25/go-wz) library for reading WZ files, with necessary fields exported for direct access.

## Key Differences from the C++ Version

- **Sorts nodes by coordinates** - Nodes are sorted by X coordinate within name-based groups for optimal ordering
- Written in Go for better cross-platform support and memory safety
- Uses LZ4 compression for bitmaps and audio data

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/ErwinsExpertise/go-wztonx-converter/releases).

Extract and run:
```bash
# Linux/macOS
tar -xzf wztonx-converter_*.tar.gz
./wztonx-converter --help

# Windows
# Extract the .zip file and run wztonx-converter.exe
```

### From Source

```bash
git clone https://github.com/ErwinsExpertise/go-wztonx-converter
cd go-wztonx-converter
go build
```

### Go Install

```bash
go install github.com/ErwinsExpertise/go-wztonx-converter@latest
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
- `--cpuprofile <file>`: Write CPU profile to file (for performance analysis)
- `--memprofile <file>`: Write memory profile to file (for memory analysis)

### Performance Profiling

To profile the converter's performance, use the profiling flags:

```bash
# CPU profiling
./go-wztonx-converter --cpuprofile cpu.prof --client file.wz

# Memory profiling
./go-wztonx-converter --memprofile mem.prof --client file.wz

# Analyze profiles with pprof
go tool pprof cpu.prof
go tool pprof mem.prof
```

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

### Performance Optimizations

The converter includes several performance optimizations:

- **Parallel Bitmap Compression**: Bitmaps are compressed in parallel using up to 8 concurrent workers, significantly speeding up large file conversions
- **Buffered I/O**: Uses 1MB buffered writing for improved disk I/O performance
- **Progress Updates**: Shows detailed progress during conversion, including:
  - Header writing status
  - Node count and progress
  - String table progress
  - Bitmap compression progress
  - Audio file progress

This makes it easy to track conversion progress and identify if the process is stuck or just processing large amounts of data.

### Node Ordering

**Important**: This implementation sorts nodes by their coordinates and name patterns for optimal ordering. Children nodes are sorted by:
1. Name-based priority groups (sp, h_*, out*, in*, st*, market*)
2. X coordinate within each group
3. Alphabetically by name as a tiebreaker

### Compression

- Bitmap data is compressed using LZ4 or LZ4HC
- Audio data is typically already compressed (MP3, etc.) and may not need additional compression
- LZ4HC provides better compression at the cost of slower encoding

### Memory-Mapped I/O

The go-wz library uses memory-mapped I/O for efficient WZ file reading.

## Dependencies

- Forked `wz` package (based on [github.com/diamondo25/go-wz](https://github.com/diamondo25/go-wz)) - WZ file parsing with exported fields for cleaner access
- [github.com/pierrec/lz4/v4](https://github.com/pierrec/lz4) - LZ4 compression
- [github.com/edsrzf/mmap-go](https://github.com/edsrzf/mmap-go) - Memory-mapped file I/O
- [github.com/goinggo/workpool](https://github.com/goinggo/workpool) - Concurrent processing

## License

This project follows the same license as the original NoLifeStory project.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## Credits

- Original C++ implementation: [NoLifeStory by Peter Atashian](https://github.com/NoLifeDev/NoLifeStory)
- WZ parsing library: [go-wz by diamondo25](https://github.com/diamondo25/go-wz)
