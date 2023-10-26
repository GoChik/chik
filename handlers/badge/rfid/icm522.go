package rfid

import (
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

var logger = log.With().Str("package", "reader").Logger()
var IncompleteDataError = fmt.Errorf("incomplete data")

const terminatorTag = 0xfe

type ICM522 struct {
	io.ReadCloser
}

func (d *ICM522) Decode() (*MiFareTag, error) {
	var firstByte [1]byte
	for {
		l, err := d.Read(firstByte[:])
		logger.Debug().Msgf("Read %d bytes: %v", l, firstByte)
		if l != 1 {
			return nil, io.EOF
		}
		if err != nil {
			return nil, err
		}
		if firstByte[0] == terminatorTag {
			break
		}
	}

	var tag MiFareTag
	unknownField := make([]byte, 1)
	for _, field := range [][]byte{
		tag.SAK[:],
		unknownField,
		tag.ATQA[:],
		tag.ID[:],
	} {
		l, err := d.Read(field)
		logger.Debug().Msgf("Read %d bytes: %v", l, field)
		if l != len(field) {
			return nil, IncompleteDataError
		}
		if err != nil {
			return nil, err
		}
	}

	return &tag, nil
}
