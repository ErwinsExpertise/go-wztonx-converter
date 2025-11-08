# Building and Using go-wztonx-converter

## Prerequisites

- Go 1.16 or later
- Git

## Building

```bash
# Clone the repository
git clone https://github.com/ErwinsExpertise/go-wztonx-converter
cd go-wztonx-converter

# Build the binary
go build

# Or install it to your Go bin directory
go install
```

## Usage Examples

### Basic Conversion

Convert a single WZ file to NX format:

```bash
./go-wztonx-converter Base.wz
```

This creates `Base.nx` in the same directory.

### Client Mode

Include audio and bitmap data in the conversion:

```bash
./go-wztonx-converter --client Base.wz
# or
./go-wztonx-converter -c Base.wz
```

### Server Mode

Exclude audio and bitmap data (smaller output files):

```bash
./go-wztonx-converter --server Base.wz
# or
./go-wztonx-converter -s Base.wz
```

### High Compression

Use LZ4 high compression for smaller files (slower):

```bash
./go-wztonx-converter --lz4hc Base.wz
# or
./go-wztonx-converter -h Base.wz
```

### Batch Conversion

Convert all WZ files in a directory:

```bash
./go-wztonx-converter --client /path/to/maplestory/data/
```

This will recursively process all `.wz` and `.img` files in the directory.

### Combining Options

```bash
# Client mode with high compression
./go-wztonx-converter -c -h Character.wz

# Convert entire directory with high compression
./go-wztonx-converter -c -h /path/to/wz/files/
```

## Output

The tool will display progress information:

```
NoLifeWzToNx - Go Edition
Converts WZ files into NX files

Base.wz -> Base.nx
Parsing input.......Done!
Creating output.....Done!
Took 5 seconds
```

## Performance Tips

1. **Server Mode**: If you don't need images or sounds, use server mode for faster conversion
2. **Standard LZ4**: Use standard LZ4 compression (default) for faster conversion
3. **High Compression**: Use LZ4HC only if file size is critical and you can afford slower conversion
4. **Batch Processing**: Converting multiple files in one invocation is more efficient than running the tool multiple times

## Troubleshooting

### "Not a PKG1/WZ file" Error

Make sure the file is actually a WZ file. The tool supports:
- `.wz` files (WZ archives)
- `.img` files (individual WZ images)

### Memory Issues

If you encounter memory issues with large WZ files:
- Make sure you have sufficient RAM
- Try converting files individually instead of batch processing
- Use server mode if you don't need images/audio

### Unsupported Image Formats

Some DXT compressed images require external decompression libraries. The tool will:
- Log a warning for unsupported formats
- Continue processing other data
- Create an NX file with empty bitmap data for unsupported formats

## Development

### Running Tests

```bash
go test -v
```

### Running Tests with Coverage

```bash
go test -cover
```

### Building for Multiple Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o wztonx-linux-amd64

# Windows
GOOS=windows GOARCH=amd64 go build -o wztonx-windows-amd64.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o wztonx-darwin-amd64
```

## File Format Details

### NX File Structure

```
[Header - 52 bytes]
  - Magic: "PKG4"
  - Node count (4 bytes)
  - Node offset (8 bytes)
  - String count (4 bytes)
  - String offset (8 bytes)
  - Bitmap count (4 bytes)
  - Bitmap offset (8 bytes)
  - Audio count (4 bytes)
  - Audio offset (8 bytes)

[Nodes Section]
  - Array of 20-byte node structures
  - Each node contains: name ID, children info, type, data

[Strings Section]
  - Length-prefixed UTF-8 strings
  - Deduplicated for efficiency

[Bitmaps Section] (client mode only)
  - Bitmap info table (width, height, offset)
  - LZ4-compressed RGBA bitmap data

[Audio Section] (client mode only)
  - Audio info table (length, offset)
  - Raw audio data
```

### Node Ordering

**Important**: This implementation preserves the original node order from the WZ file and does NOT sort nodes. This is different from the C++ version which sorts nodes by name.

## Contributing

See the main [README.md](README.md) for contribution guidelines.
