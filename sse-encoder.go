// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Server-Sent Events
// W3C Working Draft 29 October 2009
// http://www.w3.org/TR/2009/WD-eventsource-20091029/

const ContentType = "text/event-stream;charset=utf-8"

var (
	contentType = []string{ContentType}
	noCache     = []string{"no-cache"}
)

var fieldReplacer = strings.NewReplacer(
	"\n", "\\n",
	"\r", "\\r",
)

var dataReplacer = strings.NewReplacer(
	"\n", "\ndata:",
	"\r", "\\r",
)

type Event struct {
	Event string
	Id    string //nolint:staticcheck // ST1003: public API field Id kept for backward compatibility
	Retry uint
	Data  any
}

func Encode(writer io.Writer, event Event) error {
	w := checkWriter(writer)
	if err := writeID(w, event.Id); err != nil {
		return err
	}
	if err := writeEvent(w, event.Event); err != nil {
		return err
	}
	if err := writeRetry(w, event.Retry); err != nil {
		return err
	}
	return writeData(w, event.Data)
}

func writeID(w stringWriter, id string) error {
	if len(id) > 0 {
		if _, err := w.WriteString("id:"); err != nil {
			return err
		}
		if _, err := fieldReplacer.WriteString(w, id); err != nil {
			return err
		}
		_, err := w.WriteString("\n")
		return err
	}
	return nil
}

func writeEvent(w stringWriter, event string) error {
	if len(event) > 0 {
		if _, err := w.WriteString("event:"); err != nil {
			return err
		}
		if _, err := fieldReplacer.WriteString(w, event); err != nil {
			return err
		}
		_, err := w.WriteString("\n")
		return err
	}
	return nil
}

func writeRetry(w stringWriter, retry uint) error {
	if retry > 0 {
		if _, err := w.WriteString("retry:"); err != nil {
			return err
		}
		if _, err := w.WriteString(strconv.FormatUint(uint64(retry), 10)); err != nil {
			return err
		}
		_, err := w.WriteString("\n")
		return err
	}
	return nil
}

func writeData(w stringWriter, data any) error {
	if _, err := w.WriteString("data:"); err != nil {
		return err
	}

	bData, ok := data.([]byte)
	if ok {
		if _, err := dataReplacer.WriteString(w, string(bData)); err != nil {
			return err
		}
		_, err := w.WriteString("\n\n")
		return err
	}

	switch kindOfData(data) { //nolint:exhaustive // remaining kinds need no special encoding
	case reflect.Struct, reflect.Slice, reflect.Map:
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			return err
		}
		_, err = w.WriteString("\n")
		return err
	default:
		if _, err := dataReplacer.WriteString(w, fmt.Sprint(data)); err != nil {
			return err
		}
		_, err := w.WriteString("\n\n")
		return err
	}
}

func (r Event) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	return Encode(w, r)
}

func (r Event) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	header["Content-Type"] = contentType

	if _, exist := header["Cache-Control"]; !exist {
		header["Cache-Control"] = noCache
	}
}

func kindOfData(data any) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()
	if valueType == reflect.Pointer {
		valueType = value.Elem().Kind()
	}
	return valueType
}
