package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"io"
	"os"
)

// util package to compute hash from file size

type Result struct {
	Size   int64
	SHA256 string
	CRC32C uint32
}

func Compute(path string) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()

	h := sha256.New()
	crc := crc32.New(crc32.MakeTable(crc32.Castagnoli))

	// Copy once, update both digests
	n, err := io.Copy(io.MultiWriter(h, crc), f)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Size:   n,
		SHA256: hex.EncodeToString(h.Sum(nil)),
		CRC32C: crc.Sum32(),
	}, nil
}
