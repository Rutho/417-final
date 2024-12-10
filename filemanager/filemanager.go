package filemanager

import (
	"bytes"
	"crypto/sha1"
	"io"
	"os"
	"sync"
)

//example:
//note: if a file named "abc" already exists, it will be truncated (emptied)
// f := filemanager.Open("abc")
// n, err := f.WriteOffset(5, []byte{4, 5, 6})
// output: abc = [0 0 0 0 0 4 5 6], n = 3, err = nil
// buf, n, err := f.ReadPieceOffset(1, 5, 0, 5)
// output: buf = [4, 5, 6, 0, 0], n = 3, err = io.EOF

type FileManager struct {
	Filename string
	fdLock   *sync.RWMutex
	fd       *os.File
	infoHash []byte
	writeCh  chan []byte
}

func Open(filename string, infoHash []byte) (f *FileManager) {
	f = new(FileManager)
	fd, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	f.Filename = filename
	f.fd = fd
	f.fdLock = new(sync.RWMutex)
	f.infoHash = infoHash
	f.writeCh = make(chan []byte, 200)
	return f
}

// Reads n bytes of piece of index pieceIdx starting at start_offset relative to start of pieceIdx (pieceIdx * piece_size + start_offset)
// Returns a byte slice of the data read, the number of bytes actually read, and error.
// Note: if we reach the end of file before reading n bytes, byte slice will pad the remaining space with 0s to reach n bytes
// nRead will output the number of bytes we actually read (not counting the padding), and error will return io.EOF.
// Example at the start of this file shows this case
func (f *FileManager) ReadPieceOffset(pieceIdx int, pieceSize int, startOffset int, n uint32) ([]byte, int, error) {
	offset := (int64(pieceIdx)*int64(pieceSize) + int64(startOffset))
	buf, nRead, err := f.ReadOffset(offset, n)
	return buf, nRead, err
}

func (f *FileManager) ReadOffset(offset int64, n uint32) ([]byte, int, error) {
	buf := make([]byte, n)
	f.fdLock.RLock()
	defer f.fdLock.RUnlock()
	nRead, err := f.fd.ReadAt(buf, offset)
	return buf, nRead, err
}

// treat start_offset as starting from 0. So to start writing the first byte of a piece, you do start_offset = 0
func (f *FileManager) WritePieceOffset(pieceIdx int, pieceSize int, startOffset int, buf []byte) (int, error) {
	offset := (int64(pieceIdx)*int64(pieceSize) + int64(startOffset))
	return f.WriteOffset(offset, buf)
}

func (f *FileManager) WriteOffset(offset int64, buf []byte) (int, error) {
	f.fdLock.Lock()
	defer f.fdLock.Unlock()
	return f.fd.WriteAt(buf, offset)
}

func (f *FileManager) getPieceInfoHash(pieceIdx int) []byte {
	hashLengthBytes := 20
	startHashIdx := pieceIdx * hashLengthBytes
	endHashIdx := (pieceIdx + 1) * hashLengthBytes
	return f.infoHash[startHashIdx:endHashIdx]
}

func (f *FileManager) IsPieceDownloaded(pieceIdx int, pieceSize int) bool {
	buf, n, err := f.ReadPieceOffset(pieceIdx, pieceSize, 0, (uint32(pieceSize)))
	if err == io.EOF {
		buf = buf[:n]
	}
	pieceInfoHash := f.getPieceInfoHash(pieceIdx)
	h := sha1.New()
	sha1Hash := h.Sum(buf)
	return bytes.Equal(sha1Hash, pieceInfoHash)
}
