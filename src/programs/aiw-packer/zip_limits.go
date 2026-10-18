package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

const maxZipMetadata uint64 = 64 << 20

// Count central-directory records before archive/zip allocates their in-memory
// representation. Do not trust the end record's declared count alone.
func checkZipDirectory(filename string) (result error) {
	f, err := os.Open(filename)
	if err != nil { return err }
	defer func() { result = errors.Join(result, f.Close()) }()
	info, err := f.Stat()
	if err != nil { return err }
	size := info.Size()
	tailSize := int64(22+65535)
	if size < tailSize { tailSize = size }
	tail := make([]byte, int(tailSize))
	if _, err := f.ReadAt(tail, size-tailSize); err != nil { return err }
	end := -1
	for i := len(tail)-22; i >= 0; i-- {
		if bytes.Equal(tail[i:i+4], []byte{'P','K',5,6}) && i+22+int(binary.LittleEndian.Uint16(tail[i+20:i+22])) <= len(tail) {
			// Match the standard reader's last plausible EOCD, then reject
			// trailing data rather than validating a different earlier EOCD.
			if i+22+int(binary.LittleEndian.Uint16(tail[i+20:i+22])) != len(tail) { return fmt.Errorf("data after ZIP end record is not supported") }
			end = i
			break
		}
	}
	if end < 0 { return fmt.Errorf("ZIP end record is missing") }
	e := tail[end:]
	if binary.LittleEndian.Uint16(e[4:6]) != 0 || binary.LittleEndian.Uint16(e[6:8]) != 0 { return fmt.Errorf("multi-disk ZIP is not supported") }
	count := uint64(binary.LittleEndian.Uint16(e[10:12]))
	if binary.LittleEndian.Uint16(e[8:10]) != binary.LittleEndian.Uint16(e[10:12]) { return fmt.Errorf("inconsistent ZIP directory count") }
	length := uint64(binary.LittleEndian.Uint32(e[12:16]))
	offset := uint64(binary.LittleEndian.Uint32(e[16:20]))
	endOffset := uint64(size-tailSize+int64(end))
	// Include archive/zip's directorySize==0xffff compatibility condition so
	// both readers either use the same ZIP64 end record or reject the layout.
	needsZIP64 := count == 65535 || length == 0xffffffff || offset == 0xffffffff
	if needsZIP64 || length == 0xffff {
		if endOffset < 20 { return fmt.Errorf("ZIP64 locator is missing") }
		locator := make([]byte, 20)
		if _, err := f.ReadAt(locator, int64(endOffset-20)); err != nil { return err }
		if !bytes.Equal(locator[:4], []byte{'P','K',6,7}) {
			if needsZIP64 { return fmt.Errorf("ZIP64 locator is missing") }
		} else {
		if binary.LittleEndian.Uint32(locator[4:8]) != 0 || binary.LittleEndian.Uint32(locator[16:20]) != 1 { return fmt.Errorf("unsupported ZIP64 locator") }
		zoffset := binary.LittleEndian.Uint64(locator[8:16])
		if zoffset > endOffset-20 || endOffset-20-zoffset < 56 { return fmt.Errorf("invalid ZIP64 end offset") }
		z := make([]byte, 56)
		if _, err := f.ReadAt(z, int64(zoffset)); err != nil { return err }
		if !bytes.Equal(z[:4], []byte{'P','K',6,6}) || binary.LittleEndian.Uint32(z[16:20]) != 0 || binary.LittleEndian.Uint32(z[20:24]) != 0 { return fmt.Errorf("unsupported ZIP64 end record") }
		recordLength := binary.LittleEndian.Uint64(z[4:12])
		if recordLength < 44 || recordLength > endOffset-20-zoffset-12 || zoffset+12+recordLength != endOffset-20 { return fmt.Errorf("inconsistent ZIP64 end record range") }
		count = binary.LittleEndian.Uint64(z[32:40])
		if count != binary.LittleEndian.Uint64(z[24:32]) { return fmt.Errorf("inconsistent ZIP64 directory count") }
		length, offset = binary.LittleEndian.Uint64(z[40:48]), binary.LittleEndian.Uint64(z[48:56])
		endOffset = zoffset
		}
	}
	if count > maxEntries { return fmt.Errorf("ZIP exceeds %d entries", maxEntries) }
	if length > maxZipMetadata { return fmt.Errorf("ZIP directory metadata exceeds 64 MiB") }
	if offset > endOffset || length != endOffset-offset { return fmt.Errorf("ZIP central directory must exactly end at its end record") }
	var actual uint64
	for cursor := offset; cursor < offset+length; {
		if offset+length-cursor < 46 { return fmt.Errorf("truncated ZIP directory entry") }
		header := make([]byte, 46)
		if _, err := f.ReadAt(header, int64(cursor)); err != nil { return err }
		if !bytes.Equal(header[:4], []byte{'P','K',1,2}) { return fmt.Errorf("invalid ZIP directory entry") }
		nameSize := uint64(binary.LittleEndian.Uint16(header[28:30]))
		if nameSize > maxPathBytes { return fmt.Errorf("ZIP raw filename exceeds 4096 bytes") }
		recordSize := uint64(46)+nameSize+uint64(binary.LittleEndian.Uint16(header[30:32]))+uint64(binary.LittleEndian.Uint16(header[32:34]))
		if recordSize > offset+length-cursor { return fmt.Errorf("truncated ZIP directory data") }
		cursor += recordSize
		actual++
		if actual > maxEntries { return fmt.Errorf("ZIP exceeds %d actual entries", maxEntries) }
	}
	if actual != count { return fmt.Errorf("ZIP entry count does not match directory") }
	return nil
}
