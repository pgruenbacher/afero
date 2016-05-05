package gcs

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"

	"google.golang.org/appengine/aetest"
	"google.golang.org/cloud/storage"
)

type storer interface {
	Get(name string)
	Create(name string)
}

type gcs struct {
	bucket   *storage.BucketHandle
	database storer
}

type fakeContext struct{}

// Deadline returns the time when work done on behalf of this context
// should be canceled.  Deadline returns ok==false when no deadline is
// set.  Successive calls to Deadline return the same results.
func (c fakeContext) Deadline() (time.Time, bool) { return time.Time{}, false }

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.  Done may return nil if this context can
// never be canceled.  Successive calls to Done return the same value.
//
// WithCancel arranges for Done to be closed when cancel is called;
// WithDeadline arranges for Done to be closed when the deadline
// expires; WithTimeout arranges for Done to be closed when the timeout
// elapses.
func (c fakeContext) Done() (done <-chan struct{}) { return }

// Err returns a non-nil error value after Done is closed.  Err returns
// Canceled if the context was canceled or DeadlineExceeded if the
// context's deadline passed.  No other values for Err are defined.
// After Done is closed, successive calls to Err return the same value.
func (c fakeContext) Err() (err error) { return }

// Value returns the value associated with this context for key, or nil
// if no value is associated with key.  Successive calls to Value with
// the same key returns the same result.
//
// Use context values only for request-scoped data that transits
// processes and API boundaries, not for passing optional parameters to
// functions.
func (c fakeContext) Value(key interface{}) interface{} { return nil }

type writeableFile struct {
	*mem.File
	wc *storage.Writer
	//unfile
}

func (w writeableFile) Close() (err error) {
	defer func() {
		if err = w.wc.Close(); err != nil {
			log.Println(err)
		}
	}()
	if err = w.File.Close(); err != nil {
		return err
	}
	if err = w.File.Open(); err != nil {
		return err
	}
	defer w.File.Close()
	if _, err = io.Copy(w.wc, w.File); err != nil {
		return err
	}
	return err
}

type readableFile struct {
	// *mem.File
	r *storage.Reader
	unfile
	//unfile
}

func (w readableFile) Read(p []byte) (n int, err error) {
	return w.r.Read(p)
}

func (w readableFile) Close() error {
	return w.r.Close()
}

func (g gcs) createFile(filename string) (f afero.File, err error) {
	ctx, _, err := aetest.NewContext()
	if err != nil {
		return f, err
	}
	wc := g.bucket.Object(filename).NewWriter(ctx)
	//	Attributes can be set on the object by modifying the returned Writer's
	//	ObjectAttrs field before the first call to Write. If no ContentType
	//	attribute is specified, the content type will be automatically sniffed using
	//	net/http.DetectContentType
	//wc.ContentType = ""

	// we return an in-memory file so that it has seek methods
	memdata, err := mem.CreateFile(filename), nil
	if err != nil {
		return f, err
	}
	return writeableFile{
		File: mem.NewFileHandle(memdata),
		wc:   wc,
	}, nil
}

func (g gcs) openFile(filename string) (f afero.File, err error) {
	var r *storage.Reader
	ctx, _, err := aetest.NewContext()
	if err != nil {
		return f, err
	}
	if r, err = g.bucket.Object(filename).NewReader(ctx); err != nil {
		return f, err
	}
	return readableFile{
		r: r,
	}, nil
}

// Create creates a file in the filesystem, returning the file and an
// error, if any happens.
func (g gcs) Create(name string) (f afero.File, err error) {
	return g.createFile(name)
}

// Mkdir creates a directory in the filesystem, return an error if any
// happens.
func (g gcs) Mkdir(name string, perm os.FileMode) (err error) {
	return
}

// MkdirAll creates a directory path and all parents that does not exist
// yet.
func (g gcs) MkdirAll(path string, perm os.FileMode) (err error) {
	return
}

// Open opens a file, returning it or an error, if any happens.
func (g gcs) Open(name string) (f afero.File, err error) {
	return g.openFile(name)
}

// OpenFile opens a file using the given flags and the given mode.
func (g gcs) OpenFile(name string, flag int, perm os.FileMode) (f afero.File, err error) {
	return
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (g gcs) Remove(name string) (err error) {
	return
}

// RemoveAll removes a directory path and all any children it contains. It
// does not fail if the path does not exist (return nil).
func (g gcs) RemoveAll(path string) (err error) {
	return
}

// Rename renames a file.
func (g gcs) Rename(oldname, newname string) (err error) {
	return
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (g gcs) Stat(name string) (info os.FileInfo, err error) {
	return
}

// The name of this FileSystem
func (g gcs) Name() (name string) {
	return
}

//Chmod changes the mode of the named file to mode.
func (g gcs) Chmod(name string, mode os.FileMode) (err error) {
	return
}

//Chtimes changes the access and modification times of the named file
func (g gcs) Chtimes(name string, atime time.Time, mtime time.Time) (err error) {
	return
}

type unfile struct {
}

func (f unfile) Close() { return }

func (f unfile) Write(p []byte) (n int, err error) {
	return
}

func (f unfile) WriteAt(p []byte, off int64) (n int, err error) {
	return
}

func (f unfile) Read(p []byte) (n int, err error) { return }

func (f unfile) ReadAt(p []byte, off int64) (n int, err error) {
	return
}

func (f unfile) Seek(offset int64, whence int) (n int64, err error) { return }

//
//
//	io.Closer
//	io.Reader
//	io.ReaderAt
//	io.Seeker
//	io.Writer
//	io.WriterAt

func (f unfile) Name() (name string) {
	return
}
func (f unfile) Readdir(count int) (info []os.FileInfo, err error) {
	return
}
func (f unfile) Readdirnames(n int) (s []string, err error) {
	return
}
func (f unfile) Stat() (info os.FileInfo, err error) {
	return
}
func (f unfile) Sync() (err error) {
	return
}
func (f unfile) Truncate(size int64) (err error) {
	return
}
func (f unfile) WriteString(s string) (ret int, err error) {
	return
}
