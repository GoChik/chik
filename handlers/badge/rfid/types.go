package rfid

type MiFareTag struct {
	SAK  [1]byte
	ATQA [2]byte
	ID   [4]byte
}

type Decoder interface {
	Decode() (*MiFareTag, error)
	Close() error
}
