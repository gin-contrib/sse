// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sse

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testFooKey        = "foo"
	testBarKey        = "bar"
	testNewMessage    = "new_message"
	testBenchmarkData = "hi! how are you? I am fine. this is a long stupid message!!!"
)

func TestEncodeOnlyData(t *testing.T) {
	w := new(bytes.Buffer)
	event := Event{
		Data: "junk\n\njk\nid:fake",
	}
	err := Encode(w, event)
	require.NoError(t, err)
	assert.Equal(t, `data:junk
data:
data:jk
data:id:fake

`, w.String())

	decoded, _ := Decode(w)
	assert.Equal(t, "message", decoded[0].Event)
	assert.Equal(t, decoded[0].Data, []Event{event}[0].Data)
}

func TestEncodeWithEvent(t *testing.T) {
	w := new(bytes.Buffer)
	event := Event{
		Event: "t\n:<>\r\test",
		Data:  "junk\n\njk\nid:fake",
	}
	err := Encode(w, event)
	require.NoError(t, err)
	assert.Equal(t, `event:t\n:<>\r	est
data:junk
data:
data:jk
data:id:fake

`, w.String())

	decoded, _ := Decode(w)
	assert.Equal(t, "t\\n:<>\\r\test", decoded[0].Event)
	assert.Equal(t, decoded[0].Data, []Event{event}[0].Data)
}

func TestEncodeWithId(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Id:   "t\n:<>\r\test",
		Data: "junk\n\njk\nid:fa\rke",
	})
	require.NoError(t, err)
	assert.Equal(t, `id:t\n:<>\r	est
data:junk
data:
data:jk
data:id:fa\rke

`, w.String())
}

func TestEncodeWithRetry(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Retry: 11,
		Data:  "junk\n\njk\nid:fake\n",
	})
	require.NoError(t, err)
	assert.Equal(t, `retry:11
data:junk
data:
data:jk
data:id:fake
data:

`, w.String())
}

func TestEncodeWithEverything(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "abc",
		Id:    "12345",
		Retry: 10,
		Data:  "some data",
	})
	require.NoError(t, err)
	assert.Equal(t, "id:12345\nevent:abc\nretry:10\ndata:some data\n\n", w.String())
}

func TestEncodeMap(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a map",
		Data: map[string]any{
			testFooKey: "b\n\rar",
			testBarKey: "id: 2",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "event:a map\ndata:{\"bar\":\"id: 2\",\"foo\":\"b\\n\\rar\"}\n\n", w.String())
}

func TestEncodeSlice(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a slice",
		Data:  []any{1, "text", map[string]any{testFooKey: testBarKey}},
	})
	require.NoError(t, err)
	assert.Equal(t, "event:a slice\ndata:[1,\"text\",{\"foo\":\"bar\"}]\n\n", w.String())
}

func TestEncodeStruct(t *testing.T) {
	myStruct := struct {
		A int
		B string `json:"value"`
	}{1, "number"}

	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a struct",
		Data:  myStruct,
	})
	require.NoError(t, err)
	assert.Equal(t, "event:a struct\ndata:{\"A\":1,\"value\":\"number\"}\n\n", w.String())

	w.Reset()
	err = Encode(w, Event{
		Event: "a struct",
		Data:  &myStruct,
	})
	require.NoError(t, err)
	assert.Equal(t, "event:a struct\ndata:{\"A\":1,\"value\":\"number\"}\n\n", w.String())
}

func TestEncodeInteger(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "an integer",
		Data:  1,
	})
	require.NoError(t, err)
	assert.Equal(t, "event:an integer\ndata:1\n\n", w.String())
}

func TestEncodeFloat(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "Float",
		Data:  1.5,
	})
	require.NoError(t, err)
	assert.Equal(t, "event:Float\ndata:1.5\n\n", w.String())
}

func TestEncodeStream(t *testing.T) {
	w := new(bytes.Buffer)

	_ = Encode(w, Event{
		Event: "float",
		Data:  1.5,
	})

	_ = Encode(w, Event{
		Id:   "123",
		Data: map[string]any{testFooKey: testBarKey, testBarKey: testFooKey},
	})

	_ = Encode(w, Event{
		Id:    "124",
		Event: "chat",
		Data:  "hi! dude",
	})
	assert.Equal(t, "event:float\ndata:1.5\n\n"+
		"id:123\ndata:{\"bar\":\"foo\",\"foo\":\"bar\"}\n\n"+
		"id:124\nevent:chat\ndata:hi! dude\n\n", w.String())
}

func TestRenderSSE(t *testing.T) {
	w := httptest.NewRecorder()

	err := (Event{
		Event: "msg",
		Data:  "hi! how are you?",
	}).Render(w)

	require.NoError(t, err)
	assert.Equal(t, "event:msg\ndata:hi! how are you?\n\n", w.Body.String())
	assert.Equal(t, "text/event-stream;charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

func BenchmarkResponseWriter(b *testing.B) {
	w := httptest.NewRecorder()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = (Event{
			Event: testNewMessage,
			Data:  testBenchmarkData,
		}).Render(w)
	}
}

func BenchmarkFullSSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Encode(buf, Event{
			Event: testNewMessage,
			Id:    "13435",
			Retry: 10,
			Data:  testBenchmarkData,
		})
		buf.Reset()
	}
}

func BenchmarkNoRetrySSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Encode(buf, Event{
			Event: testNewMessage,
			Id:    "13435",
			Data:  testBenchmarkData,
		})
		buf.Reset()
	}
}

func BenchmarkSimpleSSE(b *testing.B) {
	buf := new(bytes.Buffer)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Encode(buf, Event{
			Event: testNewMessage,
			Data:  testBenchmarkData,
		})
		buf.Reset()
	}
}
