package filemanager

import (
	"os"
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
	fd       *os.File
}

func Open(filename string) (f *FileManager) {
	f = new(FileManager)
	fd, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	f.Filename = filename
	f.fd = fd
	return f
}

// Reads n bytes of piece of index pieceIdx starting at start_offset relative to start of pieceIdx (pieceIdx * piece_size + start_offset)
// Returns a byte slice of the data read, the number of bytes actually read, and error.
// Note: if we reach the end of file before reading n bytes, byte slice will pad the remaining space with 0s to reach n bytes
// nRead will output the number of bytes we actually read (not counting the padding), and error will return io.EOF.
// Example at the start of this file shows this case
func (f *FileManager) ReadPieceOffset(pieceIdx int, piece_size int, start_offset int, n uint32) ([]byte, int, error) {
	offset := (int64(pieceIdx)*int64(piece_size) + int64(start_offset))
	buf, nRead, err := f.ReadOffset(offset, n)
	return buf, nRead, err
}

func (f *FileManager) ReadOffset(offset int64, n uint32) ([]byte, int, error) {
	buf := make([]byte, n)
	nRead, err := f.fd.ReadAt(buf, offset)
	return buf, nRead, err
}

// treat start_offset as starting from 0. So to start writing the first byte of a piece, you do start_offset = 0
func (f *FileManager) WritePieceOffset(pieceIdx int, piece_size int, start_offset int, buf []byte) (int, error) {
	offset := (int64(pieceIdx)*int64(piece_size) + int64(start_offset))
	return f.WriteOffset(offset, buf)
}

func (f *FileManager) WriteOffset(offset int64, buf []byte) (int, error) {
	return f.fd.WriteAt(buf, offset)
}
