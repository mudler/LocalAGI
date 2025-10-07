package stdio

// This is partly re-adapted from https://github.com/modelcontextprotocol/go-sdk/tree/main/internal/jsonrpc2
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// A StdioTransport is a [Transport] that communicates over stdin/stdout using
// newline-delimited JSON.
type StdioTransport struct {
	reader *readerCloser
	writer *writerCloser
}

func NewStdioTransport(client *Client, pid string) (*StdioTransport, error) {
	reader, writer, err := client.GetProcessIO(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get process IO: %w", err)
	}
	writerCloser := &writerCloser{
		Writer: writer,
		pid:    pid,
		client: client,
	}
	readerCloser := &readerCloser{
		Reader: reader,
		pid:    pid,
		client: client,
	}
	return &StdioTransport{
		reader: readerCloser,
		writer: writerCloser,
	}, nil
}

type writerCloser struct {
	io.Writer
	pid    string
	client *Client
}

func (w *writerCloser) Write(p []byte) (n int, err error) {
	return w.Writer.Write(p)
}

func (w *writerCloser) Close() error {
	return w.client.StopProcess(w.pid)
}

type readerCloser struct {
	io.Reader
	pid    string
	client *Client
}

func (r *readerCloser) Read(p []byte) (n int, err error) {
	return r.Reader.Read(p)
}

func (r *readerCloser) Close() error {
	return r.client.StopProcess(r.pid)
}

// Connect implements the [Transport] interface.
func (t *StdioTransport) Connect(context.Context) (mcp.Connection, error) {
	return newIOConn(rwc{
		rc: t.reader,
		wc: t.writer,
	}), nil
}

// A rwc binds an io.ReadCloser and io.WriteCloser together to create an
// io.ReadWriteCloser.
type rwc struct {
	rc io.ReadCloser
	wc io.WriteCloser
}

func (r rwc) Read(p []byte) (n int, err error) {
	return r.rc.Read(p)
}

func (r rwc) Write(p []byte) (n int, err error) {
	return r.wc.Write(p)
}

func (r rwc) Close() error {
	return errors.Join(r.rc.Close(), r.wc.Close())
}

// An ioConn is a transport that delimits messages with newlines across
// a bidirectional stream, and supports jsonrpc.2 message batching.
//
// See https://github.com/ndjson/ndjson-spec for discussion of newline
// delimited JSON.
//
// See [msgBatch] for more discussion of message batching.
type ioConn struct {
	protocolVersion string // negotiated version, set during session initialization.

	writeMu sync.Mutex         // guards Write, which must be concurrency safe.
	rwc     io.ReadWriteCloser // the underlying stream

	// incoming receives messages from the read loop started in [newIOConn].
	incoming <-chan msgOrErr

	// If outgoiBatch has a positive capacity, it will be used to batch requests
	// and notifications before sending.
	outgoingBatch []jsonrpc.Message

	// Unread messages in the last batch. Since reads are serialized, there is no
	// need to guard here.
	queue []jsonrpc.Message

	// batches correlate incoming requests to the batch in which they arrived.
	// Since writes may be concurrent to reads, we need to guard this with a mutex.
	batchMu sync.Mutex
	batches map[jsonrpc2ID]*msgBatch // lazily allocated

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error
}

type msgOrErr struct {
	msg json.RawMessage
	err error
}

func newIOConn(rwc io.ReadWriteCloser) *ioConn {
	var (
		incoming = make(chan msgOrErr)
		closed   = make(chan struct{})
	)
	// Start a goroutine for reads, so that we can select on the incoming channel
	// in [ioConn.Read] and unblock the read as soon as Close is called (see #224).
	//
	// This leaks a goroutine if rwc.Read does not unblock after it is closed,
	// but that is unavoidable since AFAIK there is no (easy and portable) way to
	// guarantee that reads of stdin are unblocked when closed.
	go func() {
		dec := json.NewDecoder(rwc)
		for {
			var raw json.RawMessage
			err := dec.Decode(&raw)
			// If decoding was successful, check for trailing data at the end of the stream.
			if err == nil {
				// Read the next byte to check if there is trailing data.
				var tr [1]byte
				if n, readErr := dec.Buffered().Read(tr[:]); n > 0 {
					// If read byte is not a newline, it is an error.
					if tr[0] != '\n' {
						err = fmt.Errorf("invalid trailing data at the end of stream")
					}
				} else if readErr != nil && readErr != io.EOF {
					err = readErr
				}
			}
			select {
			case incoming <- msgOrErr{msg: raw, err: err}:
			case <-closed:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return &ioConn{
		rwc:      rwc,
		incoming: incoming,
		closed:   closed,
	}
}

func (c *ioConn) SessionID() string { return "" }

// ID is a Request identifier, which is defined by the spec to be a string, integer, or null.
// https://www.jsonrpc.org/specification#request_object
type jsonrpc2ID struct {
	value any
}

// addBatch records a msgBatch for an incoming batch payload.
// It returns an error if batch is malformed, containing previously seen IDs.
//
// See [msgBatch] for more.
func (t *ioConn) addBatch(batch *msgBatch) error {
	t.batchMu.Lock()
	defer t.batchMu.Unlock()
	for id := range batch.unresolved {
		if _, ok := t.batches[id]; ok {
			return fmt.Errorf("%+v: batch contains previously seen request (invalid request)", id.value)
		}
	}
	for id := range batch.unresolved {
		if t.batches == nil {
			t.batches = make(map[jsonrpc2ID]*msgBatch)
		}
		t.batches[id] = batch
	}
	return nil
}

// updateBatch records a response in the message batch tracking the
// corresponding incoming call, if any.
//
// The second result reports whether resp was part of a batch. If this is true,
// the first result is nil if the batch is still incomplete, or the full set of
// batch responses if resp completed the batch.
func (t *ioConn) updateBatch(resp *jsonrpc.Response) ([]*jsonrpc.Response, bool) {
	t.batchMu.Lock()
	defer t.batchMu.Unlock()

	rpcID := jsonrpc2ID{value: resp.ID.Raw()}
	if batch, ok := t.batches[rpcID]; ok {
		idx, ok := batch.unresolved[rpcID]
		if !ok {
			panic("internal error: inconsistent batches")
		}
		batch.responses[idx] = resp
		delete(batch.unresolved, rpcID)
		delete(t.batches, rpcID)
		if len(batch.unresolved) == 0 {
			return batch.responses, true
		}
		return nil, true
	}
	return nil, false
}

type msgBatch struct {
	unresolved map[jsonrpc2ID]int
	responses  []*jsonrpc.Response
}

const (
	protocolVersion20250618 = "2025-06-18"
)

func (t *ioConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	// As a matter of principle, enforce that reads on a closed context return an
	// error.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if len(t.queue) > 0 {
		next := t.queue[0]
		t.queue = t.queue[1:]
		return next, nil
	}

	var raw json.RawMessage
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case v := <-t.incoming:
		if v.err != nil {
			return nil, v.err
		}
		raw = v.msg

	case <-t.closed:
		return nil, io.EOF
	}

	msgs, batch, err := readBatch(raw)
	if err != nil {
		return nil, err
	}
	if batch && t.protocolVersion >= protocolVersion20250618 {
		return nil, fmt.Errorf("JSON-RPC batching is not supported in %s and later (request version: %s)", protocolVersion20250618, t.protocolVersion)
	}

	t.queue = msgs[1:]

	if batch {
		var respBatch *msgBatch // track incoming requests in the batch
		for _, msg := range msgs {
			if req, ok := msg.(*jsonrpc.Request); ok {
				if respBatch == nil {
					respBatch = &msgBatch{
						unresolved: make(map[jsonrpc2ID]int),
					}
				}
				rpcID := jsonrpc2ID{value: req.ID.Raw()}
				if _, ok := respBatch.unresolved[rpcID]; ok {
					return nil, fmt.Errorf("duplicate message ID %q", req.ID)
				}
				respBatch.unresolved[rpcID] = len(respBatch.responses)
				respBatch.responses = append(respBatch.responses, nil)
			}
		}
		if respBatch != nil {
			// The batch contains one or more incoming requests to track.
			if err := t.addBatch(respBatch); err != nil {
				return nil, err
			}
		}
	}
	return msgs[0], err
}

// readBatch reads batch data, which may be either a single JSON-RPC message,
// or an array of JSON-RPC messages.
func readBatch(data []byte) (msgs []jsonrpc.Message, isBatch bool, _ error) {
	// Try to read an array of messages first.
	var rawBatch []json.RawMessage
	if err := json.Unmarshal(data, &rawBatch); err == nil {
		if len(rawBatch) == 0 {
			return nil, true, fmt.Errorf("empty batch")
		}
		for _, raw := range rawBatch {
			msg, err := jsonrpc.DecodeMessage(raw)
			if err != nil {
				return nil, true, err
			}
			msgs = append(msgs, msg)
		}
		return msgs, true, nil
	}
	// Try again with a single message.
	msg, err := jsonrpc.DecodeMessage(data)
	return []jsonrpc.Message{msg}, false, err
}

func (t *ioConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	// As in [ioConn.Read], enforce that Writes on a closed context are an error.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	// Batching support: if msg is a Response, it may have completed a batch, so
	// check that first. Otherwise, it is a request or notification, and we may
	// want to collect it into a batch before sending, if we're configured to use
	// outgoing batches.
	if resp, ok := msg.(*jsonrpc.Response); ok {
		if batch, ok := t.updateBatch(resp); ok {
			if len(batch) > 0 {
				data, err := marshalMessages(batch)
				if err != nil {
					return err
				}
				data = append(data, '\n')
				_, err = t.rwc.Write(data)
				return err
			}
			return nil
		}
	} else if len(t.outgoingBatch) < cap(t.outgoingBatch) {
		t.outgoingBatch = append(t.outgoingBatch, msg)
		if len(t.outgoingBatch) == cap(t.outgoingBatch) {
			data, err := marshalMessages(t.outgoingBatch)
			t.outgoingBatch = t.outgoingBatch[:0]
			if err != nil {
				return err
			}
			data = append(data, '\n')
			_, err = t.rwc.Write(data)
			return err
		}
		return nil
	}
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	data = append(data, '\n') // newline delimited
	_, err = t.rwc.Write(data)
	return err
}

func (t *ioConn) Close() error {
	t.closeOnce.Do(func() {
		t.closeErr = t.rwc.Close()
		close(t.closed)
	})
	return t.closeErr
}

func marshalMessages[T jsonrpc.Message](msgs []T) ([]byte, error) {
	var rawMsgs []json.RawMessage
	for _, msg := range msgs {
		raw, err := jsonrpc.EncodeMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("encoding batch message: %w", err)
		}
		rawMsgs = append(rawMsgs, raw)
	}
	return json.Marshal(rawMsgs)
}
