// +build go1.12

/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package fifo

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFrom(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ctx := context.Background()

	pipeW, err := OpenFifo(ctx, filepath.Join(dir, "fifo"), os.O_RDWR|os.O_CREATE, 0600)
	assert.NoError(t, err)
	defer pipeW.Close()

	pipeR, err := OpenFifo(ctx, filepath.Join(dir, "fifo"), os.O_RDONLY, 0600)
	assert.NoError(t, err)
	defer pipeR.Close()

	f, err := os.OpenFile(filepath.Join(dir, "data"), os.O_RDWR|os.O_CREATE, 0600)
	assert.NoError(t, err)
	defer f.Close()

	data := strings.Repeat("This is a test, this is only a test.", 100)
	_, err = f.WriteString(data)
	assert.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	done := make(chan struct{})
	go func() {
		io.Copy(buf, pipeR)
		close(done)
	}()

	_, err = f.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	n, err := pipeW.(io.ReaderFrom).ReadFrom(f)
	pipeW.Close()
	assert.NoError(t, err)

	<-done
	assert.Equal(t, int64(len(data)), n)
	assert.Equal(t, buf.String(), data)
}

func TestWriteTo(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	ctx := context.Background()

	pipeW, err := OpenFifo(ctx, filepath.Join(dir, "fifo"), os.O_RDWR|os.O_CREATE, 0600)
	assert.NoError(t, err)
	defer pipeW.Close()

	pipeR, err := OpenFifo(ctx, filepath.Join(dir, "fifo"), os.O_RDONLY, 0600)
	assert.NoError(t, err)
	defer pipeR.Close()

	f, err := os.OpenFile(filepath.Join(dir, "data"), os.O_RDWR|os.O_CREATE, 0600)
	assert.NoError(t, err)
	defer f.Close()

	data := strings.Repeat("This is a test, this is only a test.", 100)

	done := make(chan struct{})
	go func() {
		io.Copy(pipeW, strings.NewReader(data))
		pipeW.Close()
		close(done)
	}()

	_, err = f.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	n, err := pipeR.(io.WriterTo).WriteTo(f)
	pipeR.Close()
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), n)

	<-done

	_, err = f.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, f)
	assert.NoError(t, err)
	assert.Equal(t, buf.String(), data)
}
