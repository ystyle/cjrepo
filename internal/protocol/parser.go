package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	META_DATA_VERSION = 1
	TARBALL_VERSION   = 1
	HEADER_SIZE       = 5 // 1 byte version + 4 bytes size
	MAX_FILE_SIZE     = 500 * 1024 * 1024 // 500MB
)

var (
	ErrInvalidFormat  = errors.New("invalid binary format")
	ErrInvalidVersion = errors.New("unsupported version")
	ErrSizeMismatch   = errors.New("size mismatch")
	ErrIncompleteData = errors.New("incomplete data")
)

// PublishRequest represents parsed publish request data
type PublishRequest struct {
	MetaData []byte // meta-data.json content
	Tarball  []byte // .cjp file content
}

// ParsePublishData parses cjpm publish binary data
// Format: [version(1)][size(4)][data][version(1)][size(4)][data]
func ParsePublishData(data []byte) (*PublishRequest, error) {
	reader := bytes.NewReader(data)
	req := &PublishRequest{}

	// Parse meta-data section
	metaData, err := readSection(reader, META_DATA_VERSION, "meta-data")
	if err != nil {
		return nil, fmt.Errorf("failed to parse meta-data: %w", err)
	}
	req.MetaData = metaData

	// Parse tarball section
	tarball, err := readSection(reader, TARBALL_VERSION, "tarball")
	if err != nil {
		return nil, fmt.Errorf("failed to parse tarball: %w", err)
	}
	req.Tarball = tarball

	return req, nil
}

// readSection reads one data section
func readSection(reader *bytes.Reader, expectedVersion byte, sectionType string) ([]byte, error) {
	// Read version (1 byte)
	version, err := reader.ReadByte()
	if err != nil {
		if err == io.EOF {
			return nil, ErrIncompleteData
		}
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	if version != expectedVersion {
		return nil, fmt.Errorf("%w: expected version %d, got %d",
			ErrInvalidVersion, expectedVersion, version)
	}

	// Read size (4 bytes, little endian)
	var size int32
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("failed to read size: %w", err)
	}

	if size < 0 || int(size) > MAX_FILE_SIZE {
		return nil, fmt.Errorf("%w: invalid size %d (max %d)",
			ErrInvalidFormat, size, MAX_FILE_SIZE)
	}

	// Read data
	data := make([]byte, size)
	n, err := reader.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	if int32(n) != size {
		return nil, ErrSizeMismatch
	}

	return data, nil
}

// CalculateSHA256 calculates SHA256 checksum of data
func CalculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// ValidateTarballSHA256 validates tarball SHA256 checksum
func ValidateTarballSHA256(tarball []byte, expectedSHA256 string) bool {
	actualSHA256 := CalculateSHA256(tarball)
	return actualSHA256 == expectedSHA256
}
