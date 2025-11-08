# TODO and Future Improvements

## High Priority

### DXT Texture Decompression
- [ ] Integrate DXT3/DXT5 decompression
  - C++ version uses libsquish
  - Go alternatives:
    - Port libsquish to Go
    - Use CGo bindings to libsquish
    - Implement DXT decompression in pure Go
- [ ] Handle Format2 scaling (format2 == 4 means scale by 16)

## Medium Priority

### WZ Encryption Keys
- [ ] Implement AES key generation at runtime
- [ ] Support BMS, GMS, and KMS keys
- [ ] Add key selection via command-line flag

### Performance Optimizations
- [ ] Parallel processing of independent nodes
- [ ] Streaming write for large files
- [ ] Memory pool for frequent allocations
- [ ] Profile and optimize hot paths

### Error Handling
- [ ] Better error messages for corrupted WZ files
- [ ] Validation of NX output format
- [ ] Recovery from partial failures

## Low Priority

### Features
- [ ] Dry-run mode (parse without writing)
- [ ] Verbose mode with detailed progress
- [ ] JSON export of node structure
- [ ] NX to WZ converter (reverse operation)

### Code Quality
- [ ] Increase test coverage to >80%
- [ ] Add integration tests with real WZ files
- [ ] Add benchmarks for performance tracking
- [ ] Document all exported functions

### UOL (Link) Handling
- [ ] Properly resolve UOL references
- [ ] Verify link integrity
- [ ] Option to inline or preserve links

## Known Limitations

1. **DXT Compression**: Currently, DXT3 and DXT5 compressed images produce empty bitmap data. This requires porting or integrating a DXT decompression library.

2. **Unexported Fields**: Uses unsafe reflection to access unexported fields in go-wz library. A cleaner solution would be to fork go-wz and export necessary fields.

3. **Memory Usage**: Large WZ files are fully loaded into memory. Streaming processing would reduce memory footprint.

4. **Format2 Scaling**: The C++ version has special handling for format2 == 4 (16x scaling). This is not implemented.

5. **Encryption Keys**: Currently relies on go-wz's key detection. Direct key selection is not implemented.

## Dependencies to Consider

- **github.com/golang/snappy**: Alternative compression
- **CGo + libsquish**: For DXT decompression
- **github.com/disintegration/imaging**: Image processing utilities

## Architecture Improvements

- [ ] Separate concerns: WZ parsing, NX writing, image processing
- [ ] Plugin architecture for image format handlers
- [ ] Configuration file support
- [ ] Better separation of client/server logic

## Documentation

- [ ] API documentation with godoc
- [ ] More usage examples
- [ ] Performance benchmarks documentation
- [ ] Comparison with C++ version
