package autils

import (
	"crypto/md5"
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// Code inspired from mgo/bson ObjectId

// ID represents a unique request id
type ID [rawLen]byte

const (
	encodedLen = 20 // string encoded len
	decodedLen = 15 // len after base32 decoding with the padded data
	rawLen     = 12 // binary raw len

	// encoding stores a custom version of the base32 encoding with lower case
	// letters.
	encoding = "0123456789abcdefghijklmnopqrstuv"
)

// ErrInvalidID is returned when trying to unmarshal an invalid ID
var ErrInvalidID = errors.New("error: invalid ID")

// objectIDCounter is atomically incremented when generating a new ObjectId
// using NewObjectId() function. It's used as a counter part of an id.
// This id is initialized with a random value.
var objectIDCounter = randInt()

// machineId stores machine id generated once and used in subsequent calls
// to NewObjectId function.
var machineID = readMachineID()

// pid stores the current process id
var pid = os.Getpid()

// dec is the decoding map for base32 encoding
var dec [256]byte

func init() {
	for i := 0; i < len(dec); i++ {
		dec[i] = 0xFF
	}
	for i := 0; i < len(encoding); i++ {
		dec[encoding[i]] = byte(i)
	}
}

// readMachineId generates machine id and puts it into the machineId global
// variable. If this function fails to get the hostname, it will cause
// a runtime error.
func readMachineID() []byte {
	id := make([]byte, 3)
	if hostname, err := os.Hostname(); err == nil {
		hw := md5.New()
		hw.Write([]byte(hostname))
		copy(id, hw.Sum(nil))
	} else {
		// Fallback to rand number if machine id can't be gathered
		if _, randErr := rand.Reader.Read(id); randErr != nil {
			panic(fmt.Errorf("xid: cannot get hostname nor generate a random number: %v; %v", err, randErr))
		}
	}
	return id
}

// randInt generates a random uint32
func randInt() uint32 {
	b := make([]byte, 3)
	if _, err := rand.Reader.Read(b); err != nil {
		panic(fmt.Errorf("xid: cannot generate random number: %v;", err))
	}
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

// GetUUID generates a globaly unique ID
func GetUUID() ID {
	var id ID
	// Timestamp, 4 bytes, big endian
	binary.BigEndian.PutUint32(id[:], uint32(time.Now().Unix()))
	// Machine, first 3 bytes of md5(hostname)
	id[4] = machineID[0]
	id[5] = machineID[1]
	id[6] = machineID[2]
	// Pid, 2 bytes, specs don't specify endianness, but we use big endian.
	id[7] = byte(pid >> 8)
	id[8] = byte(pid)
	// Increment, 3 bytes, big endian
	i := atomic.AddUint32(&objectIDCounter, 1)
	id[9] = byte(i >> 16)
	id[10] = byte(i >> 8)
	id[11] = byte(i)
	return id
}

// UUIDFromString reads an ID from its string representation
func UUIDFromString(id string) (ID, error) {
	i := &ID{}
	err := i.UnmarshalText([]byte(id))
	return *i, err
}

// UUIDToString returns a base32 hex lowercased with no padding representation of the id (char set is 0-9, a-v).
func (id ID) UUIDToString() string {
	text := make([]byte, encodedLen)
	encode(text, id[:])
	return string(text)
}

// MarshalText implements encoding/text TextMarshaler interface
func (id ID) MarshalText() ([]byte, error) {
	text := make([]byte, encodedLen)
	encode(text, id[:])
	return text, nil
}

// encode by unrolling the stdlib base32 algorithm + removing all safe checks
func encode(dst, id []byte) {
	dst[0] = encoding[id[0]>>3]
	dst[1] = encoding[(id[1]>>6)&0x1F|(id[0]<<2)&0x1F]
	dst[2] = encoding[(id[1]>>1)&0x1F]
	dst[3] = encoding[(id[2]>>4)&0x1F|(id[1]<<4)&0x1F]
	dst[4] = encoding[id[3]>>7|(id[2]<<1)&0x1F]
	dst[5] = encoding[(id[3]>>2)&0x1F]
	dst[6] = encoding[id[4]>>5|(id[3]<<3)&0x1F]
	dst[7] = encoding[id[4]&0x1F]
	dst[8] = encoding[id[5]>>3]
	dst[9] = encoding[(id[6]>>6)&0x1F|(id[5]<<2)&0x1F]
	dst[10] = encoding[(id[6]>>1)&0x1F]
	dst[11] = encoding[(id[7]>>4)&0x1F|(id[6]<<4)&0x1F]
	dst[12] = encoding[id[8]>>7|(id[7]<<1)&0x1F]
	dst[13] = encoding[(id[8]>>2)&0x1F]
	dst[14] = encoding[(id[9]>>5)|(id[8]<<3)&0x1F]
	dst[15] = encoding[id[9]&0x1F]
	dst[16] = encoding[id[10]>>3]
	dst[17] = encoding[(id[11]>>6)&0x1F|(id[10]<<2)&0x1F]
	dst[18] = encoding[(id[11]>>1)&0x1F]
	dst[19] = encoding[(id[11]<<4)&0x1F]
}

// UnmarshalText implements encoding/text TextUnmarshaler interface
func (id *ID) UnmarshalText(text []byte) error {
	if len(text) != encodedLen {
		return ErrInvalidID
	}
	for _, c := range text {
		if dec[c] == 0xFF {
			return ErrInvalidID
		}
	}
	decode(id, text)
	return nil
}

// decode by unrolling the stdlib base32 algorithm + removing all safe checks
func decode(id *ID, src []byte) {
	id[0] = dec[src[0]]<<3 | dec[src[1]]>>2
	id[1] = dec[src[1]]<<6 | dec[src[2]]<<1 | dec[src[3]]>>4
	id[2] = dec[src[3]]<<4 | dec[src[4]]>>1
	id[3] = dec[src[4]]<<7 | dec[src[5]]<<2 | dec[src[6]]>>3
	id[4] = dec[src[6]]<<5 | dec[src[7]]
	id[5] = dec[src[8]]<<3 | dec[src[9]]>>2
	id[6] = dec[src[9]]<<6 | dec[src[10]]<<1 | dec[src[11]]>>4
	id[7] = dec[src[11]]<<4 | dec[src[12]]>>1
	id[8] = dec[src[12]]<<7 | dec[src[13]]<<2 | dec[src[14]]>>3
	id[9] = dec[src[14]]<<5 | dec[src[15]]
	id[10] = dec[src[16]]<<3 | dec[src[17]]>>2
	id[11] = dec[src[17]]<<6 | dec[src[18]]<<1 | dec[src[19]]>>4
}

// UUIDTime returns the timestamp part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) UUIDTime() time.Time {
	// First 4 bytes of ObjectId is 32-bit big-endian seconds from epoch.
	secs := int64(binary.BigEndian.Uint32(id[0:4]))
	return time.Unix(secs, 0)
}

// UUIDMachine returns the 3-byte machine id part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) UUIDMachine() []byte {
	return id[4:7]
}

// UUIDPid returns the process id part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) UUIDPid() uint16 {
	return binary.BigEndian.Uint16(id[7:9])
}

// UUIDCounter returns the incrementing value part of the id.
// It's a runtime error to call this method with an invalid id.
func (id ID) UUIDCounter() int32 {
	b := id[9:12]
	// Counter is stored as big-endian 3-byte value
	return int32(uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]))
}

// UUIDValue implements the driver.Valuer interface.
func (id ID) UUIDValue() (driver.Value, error) {
	b, err := id.MarshalText()
	return string(b), err
}

// UUIDScan implements the sql.Scanner interface.
func (id *ID) UUIDScan(value interface{}) (err error) {
	switch val := value.(type) {
	case string:
		return id.UnmarshalText([]byte(val))
	case []byte:
		return id.UnmarshalText(val)
	default:
		return fmt.Errorf("xid: scanning unsupported type: %T", value)
	}
}
