package internal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type imageListerMock func(context.Context, types.ImageListOptions) ([]types.ImageSummary, error)

func (l imageListerMock) ImageList(ctx context.Context, opts types.ImageListOptions) ([]types.ImageSummary, error) {
	return l(ctx, opts)
}

func TestImageExists(t *testing.T) {
	testCases := []struct {
		desc           string
		images         []types.ImageSummary
		listError      error
		expectedResult bool
		expectedError  error
	}{
		{
			desc:           "list error",
			listError:      errors.New("nope nope nope"),
			expectedResult: false,
			expectedError:  errors.New("unable to list images: nope nope nope"),
		},
		{
			desc:           "empty image list",
			images:         []types.ImageSummary{},
			expectedResult: false,
		},
		{
			desc: "image list",
			images: []types.ImageSummary{
				{ID: "aaaaaa"},
			},
			expectedResult: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var sentOpts types.ImageListOptions
			mock := imageListerMock(func(ctx context.Context, opts types.ImageListOptions) ([]types.ImageSummary, error) {
				sentOpts = opts
				return test.images, test.listError
			})

			res, err := ImageExists(ctx, mock, "foo")
			assert.True(t, sentOpts.All)
			assert.True(t, sentOpts.Filters.ExactMatch(imageFilterReference, "foo"))
			assert.Equal(t, test.expectedError, err)
			assert.Equal(t, test.expectedResult, res)
		})
	}
}

type imagePullerMock func(context.Context, string, types.ImagePullOptions) (io.ReadCloser, error)

func (p imagePullerMock) ImagePull(ctx context.Context, ref string, opts types.ImagePullOptions) (io.ReadCloser, error) {
	return p(ctx, ref, opts)
}

type closerMock struct {
	io.Reader

	closeFunc func() error
}

func (c *closerMock) Close() error {
	return c.closeFunc()
}

type failReader struct {
	err error
}

func (f *failReader) Read(p []byte) (n int, err error) {
	return 0, f.err
}

func TestPullImage(t *testing.T) {
	testCases := []struct {
		desc          string
		pullError     error
		shouldClose   bool
		pullReader    io.Reader
		expectedError error
	}{
		{
			desc:          "pull error",
			pullError:     errors.New("nope nope nope"),
			expectedError: errors.New("unable to pull \"foo\": nope nope nope"),
		},
		{
			desc:          "copy error",
			pullReader:    &failReader{err: errors.New("broken")},
			shouldClose:   true,
			expectedError: errors.New("unable to pull \"foo\": broken"),
		},
		{
			desc:        "success error",
			shouldClose: true,
			pullReader:  &bytes.Buffer{},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			var readerClosed bool
			pullResult := closerMock{
				Reader: test.pullReader,
				closeFunc: func() error {
					readerClosed = true
					return nil
				},
			}

			mock := imagePullerMock(func(ctx context.Context, ref string, opts types.ImagePullOptions) (io.ReadCloser, error) {
				return &pullResult, test.pullError
			})

			err := PullImage(ctx, mock, "foo")

			assert.Equal(t, test.expectedError, err)
			if test.shouldClose {
				assert.True(t, readerClosed)
			}
		})
	}
}

type imageSaverMock func(ctx context.Context, refs []string) (io.ReadCloser, error)

func (s imageSaverMock) ImageSave(ctx context.Context, refs []string) (io.ReadCloser, error) {
	return s(ctx, refs)
}

type mockWriteSeeker struct {
	io.Writer

	seek func(int64, int) (int64, error)
}

func (m mockWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	return m.seek(offset, whence)
}

func TestSaveImages(t *testing.T) {
	ctx := context.Background()

	content := []byte("test")
	refs := []string{"a", "b", "c"}

	var (
		seekCalled bool
		seekOffset int64
		seekWhence int
	)

	destBuf := &bytes.Buffer{}
	dest := mockWriteSeeker{
		Writer: destBuf,

		seek: func(offset int64, whence int) (int64, error) {
			seekCalled = true
			seekOffset = offset
			seekWhence = whence
			return 0, nil
		},
	}

	client := imageSaverMock(func(ctx context.Context, refs []string) (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(content)), nil
	})

	require.NoError(t, SaveImages(ctx, client, dest, refs))

	assert.Equal(t, content, destBuf.Bytes())
	assert.True(t, seekCalled)
	assert.EqualValues(t, 0, seekOffset)
	assert.EqualValues(t, 0, seekWhence)
}
