package net

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadFrame reads one L1J packet frame from r.
// Wire format: [2 bytes LE: total length including header][payload].
// Returns the payload bytes (without the 2-byte length header).
func ReadFrame(r io.Reader) ([]byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("read frame header: %w", err)
	}

	totalLen := int(binary.LittleEndian.Uint16(header[:]))
	payloadLen := totalLen - 2
	if payloadLen <= 0 || payloadLen > 65533 {
		return nil, fmt.Errorf("invalid frame length: %d", totalLen)
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read frame payload (%d bytes): %w", payloadLen, err)
	}
	return payload, nil
}

// WriteFrame writes one L1J packet frame to w.
// Wire format: [2 bytes LE: len(data)+2][data].
// Header and payload are written in a single Write call to avoid
// Nagle-induced delays from splitting a tiny header + payload.
func WriteFrame(w io.Writer, data []byte) error {
	totalLen := len(data) + 2
	frame := make([]byte, totalLen)
	binary.LittleEndian.PutUint16(frame[0:2], uint16(totalLen))
	copy(frame[2:], data)

	if _, err := w.Write(frame); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}
