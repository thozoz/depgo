package net

//no http range
//

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/alist-org/alist/v3/pkg/http_range"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

var buf22MB = make([]byte, 1024*1024*22)

func dummyHttpRequest(data []byte, p http_range.Range) io.ReadCloser {

	end := p.Start + p.Length - 1

	if end >= int64(len(data)) {
		end = int64(len(data))
	}

	bodyBytes := data[p.Start:end]
	return io.NopCloser(bytes.NewReader(bodyBytes))
}

func TestDownloadOrder(t *testing.T) {
	buff := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	downloader, invocations, ranges := newDownloadRangeClient(buff)
	con, partSize := 3, 3
	d := NewDownloader(func(d *Downloader) {
		d.Concurrency = con
		d.PartSize = partSize
		d.HttpClient = downloader.HttpRequest
	})

	var start, length int64 = 2, 10
	length2 := length
	if length2 == -1 {
		length2 = int64(len(buff)) - start
	}
	req := &HttpRequestParams{
		Range: http_range.Range{Start: start, Length: length},
		Size:  int64(len(buff)),
	}
	readCloser, err := d.Download(context.Background(), req)

	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
	resultBuf, err := io.ReadAll(readCloser)
	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
	if exp, a := int(length), len(resultBuf); exp != a {
		t.Errorf("expect  buffer length=%d, got %d", exp, a)
	}
	chunkSize := int(length)/partSize + 1
	if int(length)%partSize == 0 {
		chunkSize--
	}
	if e, a := chunkSize, *invocations; e != a {
		t.Errorf("expect %v API calls, got %v", e, a)
	}

	// The exact chunk boundaries are an implementation detail: sendChunkTask
	// deliberately sizes the first chunk to the remainder (so the rest of the range
	// divides evenly by partSize) because a small leading chunk lets a video player
	// start sooner. Asserting fixed boundaries made this test fail the moment that
	// heuristic was tuned, while saying nothing about whether the bytes were right.
	//
	// What actually matters is asserted instead: the chunks must cover the requested
	// range contiguously, and the assembled body must equal the source slice.
	assertRangesCover(t, *ranges, start, length)
	assertDownloadedContent(t, resultBuf, buff[start:start+length])
}

// parseRangeSpec splits a "start-length" entry produced by downloadCaptureClient.
func parseRangeSpec(t *testing.T, spec string) (start, length int64) {
	t.Helper()
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("malformed range entry %q", spec)
	}
	s, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		t.Fatalf("malformed range start in %q: %v", spec, err)
	}
	l, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		t.Fatalf("malformed range length in %q: %v", spec, err)
	}
	return s, l
}

// assertRangesCover checks that the requested ranges are contiguous, start at the
// requested offset and sum to the requested length, in order.
func assertRangesCover(t *testing.T, specs []string, wantStart, wantLength int64) {
	t.Helper()
	if len(specs) == 0 {
		t.Fatal("no ranges were requested")
	}
	var covered int64
	next := wantStart
	for _, spec := range specs {
		s, l := parseRangeSpec(t, spec)
		if s != next {
			t.Errorf("range %q starts at %d, expected %d (ranges must be contiguous)", spec, s, next)
		}
		if l <= 0 {
			t.Errorf("range %q has non-positive length", spec)
		}
		next += l
		covered += l
	}
	if covered != wantLength {
		t.Errorf("ranges cover %d bytes, expected %d", covered, wantLength)
	}
}

// assertDownloadedContent verifies the assembled body equals the requested slice,
// which is the property the chunking heuristic must never break.
func assertDownloadedContent(t *testing.T, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("downloaded content mismatch: got %v, want %v", got, want)
	}
}
func init() {
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "2006-01-02T15:04:05.999999999"
	Formatter.FullTimestamp = true
	Formatter.ForceColors = true
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debugf("Download start")
}

func TestDownloadSingle(t *testing.T) {
	buff := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	downloader, invocations, ranges := newDownloadRangeClient(buff)
	con, partSize := 1, 3
	d := NewDownloader(func(d *Downloader) {
		d.Concurrency = con
		d.PartSize = partSize
		d.HttpClient = downloader.HttpRequest
	})

	var start, length int64 = 2, 10
	req := &HttpRequestParams{
		Range: http_range.Range{Start: start, Length: length},
		Size:  int64(len(buff)),
	}

	readCloser, err := d.Download(context.Background(), req)

	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
	resultBuf, err := io.ReadAll(readCloser)
	if err != nil {
		t.Fatalf("expect no error, got %v", err)
	}
	if exp, a := int(length), len(resultBuf); exp != a {
		t.Errorf("expect  buffer length=%d, got %d", exp, a)
	}
	if e, a := 1, *invocations; e != a {
		t.Errorf("expect %v API calls, got %v", e, a)
	}

	expectRngs := []string{"2-10"}
	for _, rng := range expectRngs {
		if !slices.Contains(*ranges, rng) {
			t.Errorf("expect range %v, but absent in return", rng)
		}
	}
	if e, a := expectRngs, *ranges; len(e) != len(a) {
		t.Errorf("expect %v ranges, got %v", e, a)
	}
}

type downloadCaptureClient struct {
	mockedHttpRequest    func(params *HttpRequestParams) (*http.Response, error)
	GetObjectInvocations int

	RetrievedRanges []string

	lock sync.Mutex
}

func (c *downloadCaptureClient) HttpRequest(ctx context.Context, params *HttpRequestParams) (*http.Response, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.GetObjectInvocations++

	if &params.Range != nil {
		c.RetrievedRanges = append(c.RetrievedRanges, fmt.Sprintf("%d-%d", params.Range.Start, params.Range.Length))
	}

	return c.mockedHttpRequest(params)
}

// TestInterruptTwiceDoesNotPanic is a regression test for
// https://github.com/AlistGo/alist/issues/8071: downloader.interrupt()
// (invoked via the io.ReadCloser Download() returns, whenever
// Concurrency > 1 makes it take the chunked/MultiReadCloser path rather
// than returning the raw HTTP response body directly) called
// close(d.chunkChannel) unconditionally, which panics with "close of
// closed channel" if interrupt() -- equivalently, Close() on the
// returned ReadCloser -- runs more than once on the same instance.
// This is exactly what utils.Closers.Close() does when it appears in a
// Closers list that gets closed twice (FileStream.Close() already
// tolerates a double-close of an os.File-like closer by swallowing
// os.ErrClosed, but a closed channel panics instead of returning an
// error, so that existing tolerance doesn't help here). Close() is
// documented as safe to call more than once by the io.Closer convention
// several callers in this codebase already rely on.
//
// This calls interrupt() directly on a minimally-constructed downloader
// rather than going through the public Download() API with Concurrency
// > 1, deliberately: exercising the real concurrent chunk-fetching
// goroutines (sendChunkTask/download) hits a separate, pre-existing data
// race in that unrelated code path (confirmed present identically on
// unmodified main via `go test -race -run TestDownloadOrder`, which
// already uses Concurrency: 3) that isn't this issue's concern and would
// make -race runs of this test flake for a reason that has nothing to do
// with the bug being fixed here.
func TestInterruptTwiceDoesNotPanic(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	d := &downloader{
		ctx:          ctx,
		cancel:       cancel,
		chunkChannel: make(chan chunk, 1),
		params:       &HttpRequestParams{Range: http_range.Range{Length: 0}},
	}

	if err := d.interrupt(); err != nil {
		t.Fatalf("first interrupt() returned an error: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second interrupt() panicked: %v", r)
		}
	}()
	if err := d.interrupt(); err != nil {
		t.Fatalf("second interrupt() returned an error: %v", err)
	}
}

func newDownloadRangeClient(data []byte) (*downloadCaptureClient, *int, *[]string) {
	capture := &downloadCaptureClient{}

	capture.mockedHttpRequest = func(params *HttpRequestParams) (*http.Response, error) {
		start, fin := params.Range.Start, params.Range.Start+params.Range.Length
		if params.Range.Length == -1 || fin >= int64(len(data)) {
			fin = int64(len(data))
		}
		bodyBytes := data[start:fin]

		header := &http.Header{}
		header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, fin-1, len(data)))
		return &http.Response{
			Body:          io.NopCloser(bytes.NewReader(bodyBytes)),
			Header:        *header,
			ContentLength: int64(len(bodyBytes)),
		}, nil
	}

	return capture, &capture.GetObjectInvocations, &capture.RetrievedRanges
}
