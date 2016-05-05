package gcs

import (
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/spf13/afero"

	"google.golang.org/appengine/aetest"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/memcache"
	"google.golang.org/cloud/storage"
)

func TestCreate(t *testing.T) {
	var err error
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	var bucketName string
	if bucketName, err = file.DefaultBucketName(ctx); err != nil {
		t.Fatal(err)
	}
	t.Log("default bucket", bucketName)
	var client *storage.Client
	if client, err = storage.NewClient(ctx); err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	fs := gcs{
		bucket: client.Bucket(bucketName),
	}
	fmt.Println("------ Creating testfile ----")
	var f afero.File
	if f, err = fs.Create("testfile"); err != nil {
		t.Fatal(err)
	}
	secondContent := "second create"
	if _, err = io.WriteString(f, secondContent); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	if f, err = fs.Open("testfile"); err != nil {
		t.Fatal(err)
	}
	var contents []byte
	if contents, err = ioutil.ReadAll(f); err != nil {
		t.Fatal(err)
	}
	fmt.Println("CONTENTS", contents)
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFoo(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	it := &memcache.Item{
		Key:   "some-key",
		Value: []byte("some-value"),
	}
	err = memcache.Set(ctx, it)
	if err != nil {
		t.Fatalf("Set err: %v", err)
	}
	it, err = memcache.Get(ctx, "some-key")
	if err != nil {
		t.Fatalf("Get err: %v; want no error", err)
	}
	if g, w := string(it.Value), "some-value"; g != w {
		t.Errorf("retrieved Item.Value = %q, want %q", g, w)
	}
}
