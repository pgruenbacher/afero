package gcs

import (
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"

	"google.golang.org/cloud"
	"google.golang.org/cloud/storage"
)

type storer interface {
	Get(name string)
	Create(name string)
}

type gcs struct {
	ctx      context.Context
	client   *storage.Client
	bucket   *storage.BucketHandle
	database storer
}

func New(project, bucket string) (*gcs, error) {
	var err error
	var scope = storage.ScopeFullControl
	ctx := context.Background()
	var client *storage.Client
	if client, err = storage.NewClient(ctx, cloud.WithScopes(scope)); err != nil {
		return nil, err
	}
	return &gcs{
		ctx:    ctx,
		client: client,
		bucket: client.Bucket(bucket),
	}, nil
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

type folder struct {
	name string
	g    gcs
	unfile
	cursor *string
}

func (f folder) Stat() (os.FileInfo, error) {
	// return empty objectAttrs
	return GcsFileInfo{isFolder: true, attrs: new(storage.ObjectAttrs)}, nil
}

func (f folder) Readdir(count int) (info []os.FileInfo, err error) {
	// 	Readdir reads the contents of the directory associated with file and returns a slice of up to n FileInfo values, as would be returned by Lstat, in directory order. Subsequent calls on the same file will yield further FileInfos.
	// If n > 0, Readdir returns at most n FileInfo structures. In this case, if Readdir returns an empty slice, it will return a non-nil error explaining why. At the end of a directory, the error is io.EOF.
	// If n <= 0, Readdir returns all the FileInfo from the directory in a single slice. In this case, if Readdir succeeds (reads all the way to the end of the directory), it returns the slice and a nil error. If it encounters an error before the end of the directory, Readdir returns the FileInfo read until that point and a non-nil error.

	q := &storage.Query{
		Prefix:     f.name,
		MaxResults: count,
		Versions:   false,
	}
	if f.cursor != nil {
		q.Cursor = *f.cursor
	}
	var objectList *storage.ObjectList
	if objectList, err = f.g.bucket.List(f.g.ctx, q); err != nil {
		// fmt.Println("readdir errror", err)
		return nil, os.ErrNotExist
	}
	// will allow for multiple readDirs
	if objectList.Next != nil {
		f.cursor = &objectList.Next.Cursor
	}
	infos := make([]GcsFileInfo, len(objectList.Results))
	for i, obj := range objectList.Results {
		// fmt.Println("RESULTS OBJ", obj.Name, obj.ContentType, obj.StorageClass)
		infos[i] = GcsFileInfo{false, obj}
	}
	// fmt.Println("PREFIXS", objectList.Prefixes)
	return
}

func (f folder) Readdirnames(n int) (s []string, err error) {
	infos, err := f.Readdir(n)
	if err != nil {
		return s, err
	}
	s = make([]string, len(infos))
	for i, info := range infos {
		s[i] = info.Name()
	}
	return s, nil
}

type readableFile struct {
	*mem.File
	attrs *storage.ObjectAttrs
	//r     *storage.Reader
	//unfile
	//unfile
}

func (r readableFile) Stat() (os.FileInfo, error) {
	return GcsFileInfo{attrs: r.attrs}, nil
}

func (g gcs) createFile(filename string) (f afero.File, err error) {
	//ctx, _, err := aetest.NewContext()
	//ctx := context.Background()
	//if err != nil {
	//	return f, err
	//}
	wc := g.bucket.Object(filename).NewWriter(g.ctx)
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

func (g gcs) openFolder(path string) (f afero.File, err error) {
	return folder{name: path, g: g}, nil
}

func (g gcs) open(filename string) (f afero.File, err error) {
	var r *storage.Reader
	//ctx, _, err := aetest.NewContext()
	//ctx := context.Background()
	if r, err = g.bucket.Object(filename).NewReader(g.ctx); err == storage.ErrObjectNotExist {
		// if the file doesn't exist, then try returning it as a folder path
		return g.openFolder(filename)
	} else if err != nil {
		return f, err
	}
	defer r.Close()
	var attrs *storage.ObjectAttrs
	if attrs, err = g.bucket.Object(filename).Attrs(g.ctx); err != nil {
		return f, err
	}
	memdata, err := mem.CreateFile(filename), nil
	if err != nil {
		return f, err
	}
	memfile := mem.NewFileHandle(memdata)
	//memfile.Open()
	//var n int64
	if _, err = io.Copy(memfile, r); err != nil {
		return f, err
	}
	// have it open for reading
	memfile.Open()
	return readableFile{
		File:  memfile,
		attrs: attrs,
		//r:     r,
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
	return nil
}

// MkdirAll creates a directory path and all parents that does not exist
// yet.
func (g gcs) MkdirAll(path string, perm os.FileMode) (err error) {
	return nil
}

// Open opens a file, returning it or an error, if any happens.
func (g gcs) Open(name string) (f afero.File, err error) {
	return g.open(name)
}

// OpenFile opens a file using the given flags and the given mode.
func (g gcs) OpenFile(name string, flag int, perm os.FileMode) (f afero.File, err error) {
	//file, err := m.openWrite(name)
	//if os.IsNotExist(err) && (flag&os.O_CREATE > 0) {
	//	file, err = m.Create(name)
	//}
	//if err != nil {
	//	return nil, err
	//}
	if flag == os.O_RDONLY {
		//file = mem.NewReadOnlyFileHandle(file.(*mem.File).Data())
		return g.open(name)
	}
	if flag&os.O_APPEND > 0 {
		//_, err = file.Seek(0, os.SEEK_END)
		//if err != nil {
		//	file.Close()
		//	return nil, err
		//}
	}
	if flag&(os.O_CREATE|os.O_WRONLY) > 0 {
		//err = file.Truncate(0)
		//if err != nil {
		//	file.Close()
		//	return nil, err
		//}
		return g.createFile(name)
	}
	// else just return readable
	return g.createFile(name)
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (g gcs) Remove(name string) (err error) {
	return g.bucket.Object(name).Delete(g.ctx)
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

type GcsFileInfo struct {
	isFolder bool
	attrs    *storage.ObjectAttrs
}

func (i GcsFileInfo) Name() string {
	return i.attrs.Name
}

func (i GcsFileInfo) Size() int64 {
	return i.attrs.Size
}

func (i GcsFileInfo) Mode() os.FileMode {
	return 0777
}

func (i GcsFileInfo) ModTime() time.Time {
	return i.attrs.Updated
}

func (i GcsFileInfo) IsDir() bool {
	return i.isFolder
}

func (i GcsFileInfo) Sys() interface{} {
	return nil
}

func (g gcs) SignedUrl(path, email string, key []byte) (string, error) {
	opts := &storage.SignedURLOptions{
		GoogleAccessID: email,
		PrivateKey:     key,
		Method:         "GET",
		Expires:        time.Now().Add(10 * time.Minute),
	}
	return storage.SignedURL(g.bucket, path, opts)
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (g gcs) Stat(name string) (info os.FileInfo, err error) {
	attrs, err := g.bucket.Object(name).Attrs(g.ctx)
	if err == storage.ErrObjectNotExist {
		return nil, os.ErrNotExist
	} else if err != nil {
		return nil, err
	}
	return GcsFileInfo{
		attrs: attrs,
	}, nil
}

// The name of this FileSystem
const Name = "Google-Cloud-Storage"

// The name of this FileSystem
func (g gcs) Name() (name string) {
	return Name
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

func (f unfile) Close() (err error) { return }

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
