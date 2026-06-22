// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sse

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
	// Per SSE spec (W3C), fields should have a space after the colon.
	// Empty data lines become "data: " (field name + colon + space, no value).
	expected := "data: junk\ndata: \ndata: jk\ndata: id:fake\n\n"
	assert.Equal(t, expected, w.String())

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
	assert.NoError(t, err)
	expected := "event: t\\n:<>\\r\test\ndata: junk\ndata: \ndata: jk\ndata: id:fake\n\n"
	assert.Equal(t, expected, w.String())

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
	assert.NoError(t, err)
	expected := "id: t\\n:<>\\r\test\ndata: junk\ndata: \ndata: jk\ndata: id:fa\\rke\n\n"
	assert.Equal(t, expected, w.String())
}

func TestEncodeWithRetry(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Retry: 11,
		Data:  "junk\n\njk\nid:fake\n",
	})
	assert.NoError(t, err)
	expected := "retry: 11\ndata: junk\ndata: \ndata: jk\ndata: id:fake\ndata: \n\n"
	assert.Equal(t, expected, w.String())
}

func TestEncodeWithEverything(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "abc",
		Id:    "12345",
		Retry: 10,
		Data:  "some data",
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "id: 12345\nevent: abc\nretry: 10\ndata: some data\n\n")
}

func TestEncodeMap(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a map",
		Data: map[string]interface{}{
			testFooKey: "b\n\rar",
			testBarKey: "id: 2",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a map\ndata: {\"bar\":\"id: 2\",\"foo\":\"b\\n\\rar\"}\n\n")
}

func TestEncodeSlice(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "a slice",
		Data:  []interface{}{1, "text", map[string]interface{}{testFooKey: testBarKey}},
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a slice\ndata: [1,\"text\",{\"foo\":\"bar\"}]\n\n")
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
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a struct\ndata: {\"A\":1,\"value\":\"number\"}\n\n")

	w.Reset()
	err = Encode(w, Event{
		Event: "a struct",
		Data:  &myStruct,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: a struct\ndata: {\"A\":1,\"value\":\"number\"}\n\n")
}

func TestEncodeInteger(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "an integer",
		Data:  1,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: an integer\ndata: 1\n\n")
}

func TestEncodeFloat(t *testing.T) {
	w := new(bytes.Buffer)
	err := Encode(w, Event{
		Event: "Float",
		Data:  1.5,
	})
	assert.NoError(t, err)
	assert.Equal(t, w.String(), "event: Float\ndata: 1.5\n\n")
}

func TestEncodeStream(t *testing.T) {
	w := new(bytes.Buffer)

	_ = Encode(w, Event{
		Event: "float",
		Data:  1.5,
	})

	_ = Encode(w, Event{
		Id:   "123",
		Data: map[string]interface{}{testFooKey: testBarKey, testBarKey: testFooKey},
	})

	_ = Encode(w, Event{
		Id:    "124",
		Event: "chat",
		Data:  "hi! dude",
	})
	assert.Equal(t, w.String(),
		"event: float\ndata: 1.5\n\n"+
			"id: 123\ndata: {\"bar\":\"foo\",\"foo\":\"bar\"}\n\n"+
			"id: 124\nevent: chat\ndata: hi! dude\n\n")
}

func TestRenderSSE(t *testing.T) {
	w := httptest.NewRecorder()

	err := (Event{
		Event: "msg",
		Data:  "hi! how are you?",
	}).Render(w)

	assert.NoError(t, err)
	assert.Equal(t, w.Body.String(), "event: msg\ndata: hi! how are you?\n\n")
	assert.Equal(t, w.Header().Get("Content-Type"), "text/event-stream;charset=utf-8")
	assert.Equal(t, w.Header().Get("Cache-Control"), "no-cache")
}

// TestSpecCompliance verifies that the encoder produces spec-compliant SSE output
// per W3C EventSource spec: fields should have a space after the colon.
func TestSpecCompliance(t *testing.T) {
	t.Run("data_field_has_space", func(t *testing.T) {
		w := new(bytes.Buffer)
		_ = Encode(w, Event{Data: "hello"})
		assert.Equal(t, "data: hello\n\n", w.String())
	})

	t.Run("event_field_has_space", func(t *testing.T) {
		w := new(bytes.Buffer)
		_ = Encode(w, Event{Event: "update", Data: "x"})
		assert.Contains(t, w.String(), "event: update\n")
	})

	t.Run("id_field_has_space", func(t *testing.T) {
		w := new(bytes.Buffer)
		_ = Encode(w, Event{Id: "42", Data: "x"})
		assert.Contains(t, w.String(), "id: 42\n")
	})

	t.Run("retry_field_has_space", func(t *testing.T) {
		w := new(bytes.Buffer)
		_ = Encode(w, Event{Retry: 5, Data: "x"})
		assert.Contains(t, w.String(), "retry: 5\n")
	})

	t.Run("multiline_data_continuation_has_space", func(t *testing.T) {
		w := new(bytes.Buffer)
		_ = Encode(w, Event{Data: "line1\nline2"})
		assert.Contains(t, w.String(), "data: line1\ndata: line2\n")
	})

	t.Run("round_trip_encode_decode", func(t *testing.T) {
		w := new(bytes.Buffer)
		event := Event{Event: "test", Id: "1", Data: "hello world"}
		_ = Encode(w, event)
		decoded, _ := Decode(w)
		assert.Equal(t, "test", decoded[0].Event)
		assert.Equal(t, "1", decoded[0].Id)
		assert.Equal(t, "hello world", decoded[0].Data)
	})
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