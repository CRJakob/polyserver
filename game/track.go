package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var cpIDs = map[uint8]struct{}{
	52: {},
	65: {},
	75: {},
	77: {},
}

var startIDs = map[uint8]struct{}{
	5:  {},
	91: {},
	92: {},
	93: {},
}

func hasCpOrder(id uint8) bool {
	_, ok := cpIDs[id]
	return ok
}

func hasStartOrder(id uint8) bool {
	_, ok := startIDs[id]
	return ok
}

type Environment uint8

const (
	Summer Environment = iota // 0
	Winter                    // 1
	Desert                    // 2
)

func (e Environment) String() string {
	switch e {
	case Summer:
		return "Summer"
	case Winter:
		return "Winter"
	case Desert:
		return "Desert"
	default:
		return fmt.Sprintf("Environment(%d)", e)
	}
}

// Track holds the metadata and the decoded track data.
type Track struct {
	Metadata TrackMetadata
	Data     *TrackInfo
}

type TrackMetadata struct {
	Name         string
	Author       *string // optional
	LastModified *time.Time
}

type TrackInfo struct {
	Env       Environment
	SunDir    uint8
	MinX      int32
	MinY      int32
	MinZ      int32
	DataBytes uint8
	Parts     []Part
}

type Part struct {
	ID     uint8
	Amount uint32
	Blocks []Block
}

type Block struct {
	X uint32
	Y uint32
	Z uint32

	Rotation  uint8
	Direction uint8

	Color uint8

	CpOrder    *uint16
	StartOrder *uint32
}

// DecodePolyTrack2 decodes a PolyTrack2-encoded string.
func DecodePolyTrack2(prefixedInput string) (*Track, error) {
	const prefix = "PolyTrack2"
	if !strings.HasPrefix(prefixedInput, prefix) {
		return nil, errors.New("invalid prefix")
	}

	// Remove the prefix
	input := strings.TrimPrefix(prefixedInput, prefix)
	fmt.Printf("Base62 input length: %d\n", len(input))

	// First base62 decode
	firstDecoded, err := DecodeBase62(input)
	if err != nil {
		return nil, fmt.Errorf("first base62 decode failed: %w", err)
	}
	fmt.Printf("First decoded length: %d bytes\n", len(firstDecoded))

	// First inflate - this should produce a STRING (the JS code uses to: "string")
	firstInflated, err := ZlibDecompressToString(firstDecoded)
	if err != nil {
		return nil, fmt.Errorf("first decompression failed: %w", err)
	}
	fmt.Printf("First inflated (string) length: %d\n", len(firstInflated))

	// Second base62 decode (on the string)
	secondDecoded, err := DecodeBase62(firstInflated)
	if err != nil {
		return nil, fmt.Errorf("second base62 decode failed: %w", err)
	}
	fmt.Printf("Second decoded length: %d bytes\n", len(secondDecoded))

	// Second inflate - this produces bytes
	secondInflated, err := ZlibDecompress(secondDecoded)
	if err != nil {
		return nil, fmt.Errorf("second decompression failed: %w", err)
	}
	fmt.Printf("Second inflated length: %d bytes\n", len(secondInflated))

	// Now parse the final byte array
	return parseTrackData(secondInflated)
}

func parseTrackData(buf []byte) (*Track, error) {
	pos := 0

	if len(buf) < 1 {
		return nil, errors.New("buffer too small")
	}

	// Name length + Name
	nameLen := int(buf[pos])
	pos++
	if len(buf) < pos+nameLen {
		return nil, errors.New("buffer too small for name")
	}
	name := string(buf[pos : pos+nameLen])
	fmt.Printf("name: %q\n", name)
	pos += nameLen

	// Author length + Author (optional)
	if len(buf) < pos+1 {
		return nil, errors.New("buffer too small for author length")
	}
	authorLen := int(buf[pos])
	pos++

	var author *string
	if authorLen > 0 {
		if len(buf) < pos+authorLen {
			return nil, errors.New("buffer too small for author")
		}
		a := string(buf[pos : pos+authorLen])
		author = &a
		pos += authorLen
	}
	fmt.Printf("author: %v\n", *author)

	// Last modified flag
	if len(buf) < pos+1 {
		return nil, errors.New("buffer too small for lastModified flag")
	}
	lmFlag := buf[pos]
	pos++

	var lastModified *time.Time
	switch lmFlag {
	case 0:
		lastModified = nil
	case 1:
		if len(buf) < pos+4 {
			return nil, errors.New("buffer too small for lastModified timestamp")
		}
		// Read little-endian uint32
		ts := uint32(buf[pos]) | uint32(buf[pos+1])<<8 | uint32(buf[pos+2])<<16 | uint32(buf[pos+3])<<24
		t := time.Unix(int64(ts), 0)
		lastModified = &t
		pos += 4
	default:
		return nil, fmt.Errorf("invalid lastModified flag: %d", lmFlag)
	}
	fmt.Printf("lastModified: %v\n", lastModified)

	// Track data (rest of the buffer)
	trackData, err := decodeTrackData(buf[pos:])
	if err != nil {
		return nil, err
	}
	return &Track{
		Metadata: TrackMetadata{
			Name:         name,
			Author:       author,
			LastModified: lastModified,
		},
		Data: trackData,
	}, nil
}

// EncodeTrack converts a TrackInfo to the binary format
func (trackInfo *TrackInfo) EncodeTrackInfo() ([]byte, error) {
	var buf bytes.Buffer

	// Write environment and sun direction
	// Using (0, d.gn)(this, a, "f") and (0, d.gn)(this, s, "f").representation
	// Assuming these are just the raw values
	buf.WriteByte(byte(trackInfo.Env))
	buf.WriteByte(trackInfo.SunDir)

	// Find min and max coordinates
	minX, minY, minZ := int32(1<<31-1), int32(1<<31-1), int32(1<<31-1)
	maxX, maxY, maxZ := int32(-1<<31), int32(-1<<31), int32(-1<<31)

	hasParts := false
	for _, part := range trackInfo.Parts {
		for _, block := range part.Blocks {
			hasParts = true
			x, y, z := int32(block.X), int32(block.Y), int32(block.Z)

			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if z < minZ {
				minZ = z
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
			if z > maxZ {
				maxZ = z
			}
		}
	}

	// If no parts or no finite values, set to 0
	if !hasParts {
		minX, minY, minZ = 0, 0, 0
		maxX, maxY, maxZ = 0, 0, 0
	}

	// Calculate ranges
	rangeX := maxX - minX + 1
	rangeY := maxY - minY + 1
	rangeZ := maxZ - minZ + 1

	// Calculate optimal byte sizes for coordinates (1-4 bytes)
	bytesX := calculateByteSize(rangeX)
	bytesY := calculateByteSize(rangeY)
	bytesZ := calculateByteSize(rangeZ)

	// Write min bounds (little-endian)
	writeInt32(&buf, minX)
	writeInt32(&buf, minY)
	writeInt32(&buf, minZ)

	// Write data bytes configuration
	// Packed as: X bytes (bits 0-1) | Y bytes (bits 2-3) | Z bytes (bits 4-5)
	dataBytes := byte(bytesX) | (byte(bytesY) << 2) | (byte(bytesZ) << 4)
	buf.WriteByte(dataBytes)

	// Write parts
	for _, part := range trackInfo.Parts {
		// Part ID
		if part.ID > 255 {
			return nil, fmt.Errorf("part ID out of range: %d", part.ID)
		}
		buf.WriteByte(part.ID)

		// Amount of blocks (little-endian uint32)
		writeUint32(&buf, part.Amount)

		// Write each block
		for _, block := range part.Blocks {
			// Write relative coordinates
			relX := int32(block.X) - minX
			relY := int32(block.Y) - minY
			relZ := int32(block.Z) - minZ

			// Write X coordinate with appropriate byte size
			writeIntWithBytes(&buf, relX, bytesX)

			// Write Y coordinate with appropriate byte size
			writeIntWithBytes(&buf, relY, bytesY)

			// Write Z coordinate with appropriate byte size
			writeIntWithBytes(&buf, relZ, bytesZ)

			// Write rotation and direction
			// Packed as: rotation (bits 0-1) | direction (bits 2-4) | (bits 5-7 unused)
			if block.Rotation > 3 {
				return nil, fmt.Errorf("rotation out of range: %d", block.Rotation)
			}
			if block.Direction > 7 {
				return nil, fmt.Errorf("direction out of range: %d", block.Direction)
			}
			metadata := block.Rotation | (block.Direction << 2)
			buf.WriteByte(metadata)

			// Write color
			buf.WriteByte(block.Color)

			// Write optional checkpoint order if this part type needs it
			if hasCpOrder(part.ID) {
				if block.CpOrder == nil {
					return nil, fmt.Errorf("checkpoint part %d missing checkpoint order", part.ID)
				}
				writeUint16(&buf, *block.CpOrder)
			}

			// Write optional start order if this part type needs it
			if hasStartOrder(part.ID) {
				if block.StartOrder == nil {
					return nil, fmt.Errorf("start part %d missing start order", part.ID)
				}
				writeUint32(&buf, *block.StartOrder)
			}
		}
	}

	return buf.Bytes(), nil
}

// decodeTrackData is a placeholder for your actual track data decoder (PolyTrack1-style)
func decodeTrackData(buf []byte) (*TrackInfo, error) {
	pos := 0

	// Check header byte
	if len(buf)-pos < 1 {
		return nil, errors.New("buffer too small for header")
	}
	header := buf[pos]
	pos++

	// Environment mapping (based on l.A in JS)
	var env Environment
	switch header {
	case 0:
		env = Summer
	case 1:
		env = Winter
	case 2:
		env = Desert
	default:
		return nil, fmt.Errorf("invalid environment: %d", header)
	}
	fmt.Printf("env: %v\n", env)

	// Sun direction
	if len(buf)-pos < 1 {
		return nil, errors.New("buffer too small for sun direction")
	}
	sunDir := buf[pos]
	fmt.Printf("sunDir: %v\n", sunDir)
	pos++

	if sunDir >= 180 {
		return nil, fmt.Errorf("invalid sun direction: %d", sunDir)
	}

	// Min bounds (3 int32 values)
	if len(buf)-pos < 12 { // 3 * 4 bytes
		return nil, errors.New("buffer too small for min bounds")
	}

	// Read little-endian int32 values
	minX := int32(buf[pos]) | int32(buf[pos+1])<<8 | int32(buf[pos+2])<<16 | int32(buf[pos+3])<<24
	pos += 4

	minY := int32(buf[pos]) | int32(buf[pos+1])<<8 | int32(buf[pos+2])<<16 | int32(buf[pos+3])<<24
	pos += 4

	minZ := int32(buf[pos]) | int32(buf[pos+1])<<8 | int32(buf[pos+2])<<16 | int32(buf[pos+3])<<24
	pos += 4
	fmt.Printf("minX: %v, minY: %v, minZ: %v\n", minX, minY, minZ)
	// Data bytes (bit packing info)
	if len(buf)-pos < 1 {
		return nil, errors.New("buffer too small for data bytes")
	}
	dataBytes := buf[pos]
	fmt.Printf("dataBytes: %v\n", dataBytes)
	pos++

	// Extract bit lengths (m, A, v from JS)
	m := int(dataBytes & 3)        // bits 0-1: X coordinate bytes
	A := int((dataBytes >> 2) & 3) // bits 2-3: Y coordinate bytes
	v := int((dataBytes >> 4) & 3) // bits 4-5: Z coordinate bytes

	// Validate ranges (1-4 as in JS)
	if m < 1 || m > 4 || A < 1 || A > 4 || v < 1 || v > 4 {
		return nil, fmt.Errorf("invalid coordinate byte lengths: X=%d, Y=%d, Z=%d", m, A, v)
	}

	trackInfo := &TrackInfo{
		Env:       env,
		SunDir:    sunDir,
		MinX:      minX,
		MinY:      minY,
		MinZ:      minZ,
		DataBytes: dataBytes,
		Parts:     make([]Part, 0),
	}

	// Parse parts
	for pos < len(buf) {
		// Part ID
		if len(buf)-pos < 1 {
			return nil, errors.New("buffer too small for part ID")
		}
		partID := buf[pos]
		pos++

		// Amount (number of blocks for this part)
		if len(buf)-pos < 4 {
			return nil, errors.New("buffer too small for part amount")
		}
		amount := uint32(buf[pos]) | uint32(buf[pos+1])<<8 | uint32(buf[pos+2])<<16 | uint32(buf[pos+3])<<24
		pos += 4

		part := Part{
			ID:     partID,
			Amount: amount,
			Blocks: make([]Block, 0, amount),
		}

		// Parse each block
		for blockIdx := 0; blockIdx < int(amount); blockIdx++ {
			block := Block{}

			// Read X coordinate
			if len(buf)-pos < m {
				return nil, fmt.Errorf("buffer too small for block X coordinate (part %d, block %d)", partID, blockIdx)
			}
			x := uint32(0)
			for i := 0; i < m; i++ {
				x |= uint32(buf[pos+i]) << (8 * i)
			}
			x += uint32(minX)
			block.X = x
			pos += m

			// Read Y coordinate
			if len(buf)-pos < A {
				return nil, fmt.Errorf("buffer too small for block Y coordinate (part %d, block %d)", partID, blockIdx)
			}
			y := uint32(0)
			for i := 0; i < A; i++ {
				y |= uint32(buf[pos+i]) << (8 * i)
			}
			y += uint32(minY)
			block.Y = y
			pos += A

			// Read Z coordinate
			if len(buf)-pos < v {
				return nil, fmt.Errorf("buffer too small for block Z coordinate (part %d, block %d)", partID, blockIdx)
			}
			z := uint32(0)
			for i := 0; i < v; i++ {
				z |= uint32(buf[pos+i]) << (8 * i)
			}
			z += uint32(minZ)
			block.Z = z
			pos += v

			// Read block metadata
			if len(buf)-pos < 1 {
				return nil, fmt.Errorf("buffer too small for block metadata (part %d, block %d)", partID, blockIdx)
			}
			metadata := buf[pos]
			pos++

			block.Rotation = metadata & 3    // bits 0-1
			direction := (metadata >> 2) & 7 // bits 2-4
			if direction > 5 {               // Assuming Direction enum has values 0-5
				return nil, fmt.Errorf("invalid direction: %d", direction)
			}
			block.Direction = direction

			// Read color
			if len(buf)-pos < 1 {
				return nil, fmt.Errorf("buffer too small for block color (part %d, block %d)", partID, blockIdx)
			}
			block.Color = buf[pos]
			pos++

			// Optional checkpoint order (for checkpoint parts)
			if hasCpOrder(partID) {
				if len(buf)-pos < 2 {
					return nil, fmt.Errorf("buffer too small for CP order (part %d, block %d)", partID, blockIdx)
				}
				cpOrder := uint16(buf[pos]) | uint16(buf[pos+1])<<8
				block.CpOrder = &cpOrder
				pos += 2
			}

			// Optional start order (for start parts)
			if hasStartOrder(partID) {
				if len(buf)-pos < 4 {
					return nil, fmt.Errorf("buffer too small for start order (part %d, block %d)", partID, blockIdx)
				}
				startOrder := uint32(buf[pos]) | uint32(buf[pos+1])<<8 | uint32(buf[pos+2])<<16 | uint32(buf[pos+3])<<24
				block.StartOrder = &startOrder
				pos += 4
			}

			part.Blocks = append(part.Blocks, block)
		}

		trackInfo.Parts = append(trackInfo.Parts, part)
	}

	return trackInfo, nil
}

func (track *Track) GetTrackID() (string, error) {
	trackInfo, err := track.Data.EncodeTrackInfo()
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	hasher.Write(trackInfo)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func DecodeTrackInfo(data []byte) (*TrackInfo, error) {
	pos := 0

	if len(data)-pos < 1 {
		return nil, fmt.Errorf("unexpected EOF (env)")
	}

	env := Environment(data[pos])
	pos++

	if len(data)-pos < 1 {
		return nil, fmt.Errorf("unexpected EOF (sun_dir)")
	}

	sunDir := data[pos]
	pos++

	if len(data)-pos < 12 {
		return nil, fmt.Errorf("unexpected EOF (min coords)")
	}

	minX := int32(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	minY := int32(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	minZ := int32(binary.LittleEndian.Uint32(data[pos:]))
	pos += 4

	if len(data)-pos < 1 {
		return nil, fmt.Errorf("unexpected EOF (packed byte)")
	}

	packed := data[pos]
	pos++

	xBytes := packed & 0b11
	yBytes := (packed >> 2) & 0b11
	zBytes := (packed >> 4) & 0b11

	if xBytes < 1 || xBytes > 4 ||
		yBytes < 1 || yBytes > 4 ||
		zBytes < 1 || zBytes > 4 {
		return nil, fmt.Errorf("invalid coordinate byte widths")
	}

	track := &TrackInfo{
		Env:       env,
		SunDir:    sunDir,
		MinX:      minX,
		MinY:      minY,
		MinZ:      minZ,
		DataBytes: packed,
		Parts:     []Part{},
	}

	// -------------------------
	// Parts Loop (matches JS)
	// -------------------------
	for pos < len(data) {

		if len(data)-pos < 1 {
			return nil, fmt.Errorf("unexpected EOF (part id)")
		}

		partID := data[pos]
		pos++

		if len(data)-pos < 4 {
			return nil, fmt.Errorf("unexpected EOF (part amount)")
		}

		amount := binary.LittleEndian.Uint32(data[pos:])
		pos += 4

		part := Part{
			ID:     partID,
			Amount: amount,
			Blocks: make([]Block, 0, amount),
		}

		// -------------------------
		// Blocks Loop
		// -------------------------
		for i := uint32(0); i < amount; i++ {

			// ---- X ----
			if len(data)-pos < int(xBytes) {
				return nil, fmt.Errorf("unexpected EOF (x)")
			}

			var x uint32
			for b := uint8(0); b < xBytes; b++ {
				x |= uint32(data[pos+int(b)]) << (8 * b)
			}
			x += uint32(minX)
			pos += int(xBytes)

			// ---- Y ----
			if len(data)-pos < int(yBytes) {
				return nil, fmt.Errorf("unexpected EOF (y)")
			}

			var y uint32
			for b := uint8(0); b < yBytes; b++ {
				y |= uint32(data[pos+int(b)]) << (8 * b)
			}
			y += uint32(minY)
			pos += int(yBytes)

			// ---- Z ----
			if len(data)-pos < int(zBytes) {
				return nil, fmt.Errorf("unexpected EOF (z)")
			}

			var z uint32
			for b := uint8(0); b < zBytes; b++ {
				z |= uint32(data[pos+int(b)]) << (8 * b)
			}
			z += uint32(minZ)
			pos += int(zBytes)

			// ---- rotation + direction ----
			if len(data)-pos < 1 {
				return nil, fmt.Errorf("unexpected EOF (rot/dir)")
			}

			rotDir := data[pos]
			pos++

			rotation := rotDir & 0b11
			direction := (rotDir >> 2) & 0b111

			// ---- color ----
			if len(data)-pos < 1 {
				return nil, fmt.Errorf("unexpected EOF (color)")
			}

			color := data[pos]
			pos++

			var cpOrder *uint16
			var startOrder *uint32

			// NOTE:
			// You must replicate JS logic:
			// if blockType in d.bK → cp_order
			// if blockType in d.l1 → start_order
			//
			// For now assume helper funcs:
			if hasCpOrder(partID) {
				if len(data)-pos < 2 {
					return nil, fmt.Errorf("unexpected EOF (cp_order)")
				}
				val := binary.LittleEndian.Uint16(data[pos:])
				cpOrder = &val
				pos += 2
			}

			if hasStartOrder(partID) {
				if len(data)-pos < 4 {
					return nil, fmt.Errorf("unexpected EOF (start_order)")
				}
				val := binary.LittleEndian.Uint32(data[pos:])
				startOrder = &val
				pos += 4
			}

			block := Block{
				X:          x,
				Y:          y,
				Z:          z,
				Rotation:   rotation,
				Direction:  direction,
				Color:      color,
				CpOrder:    cpOrder,
				StartOrder: startOrder,
			}

			part.Blocks = append(part.Blocks, block)
		}

		track.Parts = append(track.Parts, part)
	}

	return track, nil
}
