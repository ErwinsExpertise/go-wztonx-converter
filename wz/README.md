# WZ Package

This package is a fork of [github.com/diamondo25/go-wz](https://github.com/diamondo25/go-wz) with the following modifications:

## Changes from Original

1. **Exported fields in `WZSoundDX8`**:
   - `HeaderData` (was `headerData`)
   - `SoundData` (was `soundData`)
   
2. **Exported fields in `WZCanvas`**:
   - `Data` (was `data`)

These fields were exported to allow direct access to the raw WZ data without requiring unsafe reflection, resulting in cleaner and more maintainable code.

## Original License

This package maintains the license of the original go-wz library.

## Original Author

Original library by diamondo25: https://github.com/diamondo25/go-wz
