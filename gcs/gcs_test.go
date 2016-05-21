package gcs

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func getFs() (*gcs, error) {
	//var err error
	//var scope = storage.ScopeFullControl
	var bucketName string
	var projectId string
	if bucketName = os.Getenv("BUCKET_NAME"); bucketName == "" {
		return nil, errors.New("Required Env: BUCKET_NAME")
	}
	if projectId = os.Getenv("PROJECT"); projectId == "" {
		return nil, errors.New("Required Env: PROJECT")
	}
	return New(projectId, bucketName)
	//	var client *storage.Client
	//	ctx := context.Background()
	//	if client, err = storage.NewClient(ctx, cloud.WithScopes(scope)); err != nil {
	//		return nil, err
	//	}
	//	return &gcs{
	//		ctx:    ctx,
	//		client: client,
	//		bucket: client.Bucket(bucketName),
	//	}, nil
}

func TestInterface(t *testing.T) {
	var err error
	var fs *gcs
	if fs, err = getFs(); err != nil {
		t.Fatal(err)
	}
	var F afero.Fs
	F = fs
	t.Log(F)
}

func TestCreateRead(t *testing.T) {
	require := require.New(t)
	var err error
	var fs *gcs
	if fs, err = getFs(); err != nil {
		t.Fatal(err)
	}
	defer fs.client.Close()
	var f afero.File
	if f, err = fs.Create("testfile2"); err != nil {
		t.Fatal(err)
	}
	secondContent := "second create"
	if _, err = io.WriteString(f, secondContent); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	if f, err = fs.Open("testfile2"); err != nil {
		t.Fatal(err)
	}
	var contents []byte
	if contents, err = ioutil.ReadAll(f); err != nil {
		t.Fatal(err)
	}
	//fmt.Println("CONTENTS", string(contents))
	require.Equal("second create", string(contents))
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

func (fs gcs) quickCreate(name string) error {
	var err error
	var f afero.File
	if f, err = fs.Create(name); err != nil {
		return err
	}
	content := "quick create"
	if _, err = io.WriteString(f, content); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}

func TestStat(t *testing.T) {
	require := require.New(t)
	var err error
	var fs *gcs
	if fs, err = getFs(); err != nil {
		t.Fatal(err)
	}
	defer fs.client.Close()
	name := "folder1/test-stat.txt"
	if err = fs.quickCreate(name); err != nil {
		t.Fatal(err)
	}
	var info os.FileInfo
	if info, err = fs.Stat(name); err != nil {
		t.Fatal(err)
	}
	require.Equal(name, info.Name())
	require.Equal(false, info.IsDir())
}

func TestReadDir(t *testing.T) {
	// require := require.New(t)
	var err error
	var fs *gcs
	if fs, err = getFs(); err != nil {
		t.Fatal(err)
	}
	defer fs.client.Close()
	name := "folder2/test-readdir1.txt"
	if err = fs.quickCreate(name); err != nil {
		t.Fatal(err)
	}
	name = "folder2/test-readdir2.txt"
	if err = fs.quickCreate(name); err != nil {
		t.Fatal(err)
	}
	name = "folder2/nested-folder/test-readdir3.txt"
	if err = fs.quickCreate(name); err != nil {
		t.Fatal(err)
	}
	var f afero.File
	f, err = fs.Open("folder2")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}
}

//func TestBuckets(t *testing.T) {
//	var err error
//	ctx, done, err := aetest.NewContext()
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer done()
//	var bucketName string
//	if bucketName, err = file.DefaultBucketName(ctx); err != nil {
//		t.Fatal(err)
//	}
//	t.Log("default bucket", bucketName)
//	var client *storage.Client
//	if client, err = storage.NewClient(ctx); err != nil {
//		t.Fatal(err)
//	}
//	defer client.Close()
//	var bucket *storage.BucketHandle
//	filename := "test-file-1"
//	bucket = client.Bucket(bucketName)
//	wc := bucket.Object(filename).NewWriter(ctx)
//	if _, err = io.WriteString(wc, "test write value"); err != nil {
//		t.Fatal(err)
//	}
//	if err = wc.Close(); err != nil {
//		t.Fatal(err)
//	}

//var rc *storage.Reader
//if rc, err = bucket.Object(filename).NewReader(fakeContext{}); err != nil {
//	t.Fatal(err)
//}
//var content []byte
//if content, err = ioutil.ReadAll(rc); err != nil {
//	t.Fatal(err)
//}
//if err = rc.Close(); err != nil {
//	t.Fatal(err)
//}
//t.Log("CONTENT", string(content))
//}

//func TestFoo(t *testing.T) {
//	ctx, done, err := aetest.NewContext()
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer done()
//
//	it := &memcache.Item{
//		Key:   "some-key",
//		Value: []byte("some-value"),
//	}
//	err = memcache.Set(ctx, it)
//	if err != nil {
//		t.Fatalf("Set err: %v", err)
//	}
//	it, err = memcache.Get(ctx, "some-key")
//	if err != nil {
//		t.Fatalf("Get err: %v; want no error", err)
//	}
//	if g, w := string(it.Value), "some-value"; g != w {
//		t.Errorf("retrieved Item.Value = %q, want %q", g, w)
//	}
//}
