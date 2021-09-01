// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2017-present Datadog, Inc.

package decoder

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/stretchr/testify/assert"
)

// All valid whitespace characters
const whitespace = "\t\n\v\f\r\u0085\u00a0 "

func getDummyMessage(content string) *Message {
	return NewMessage([]byte(content), "info", len(content), "2018-06-14T18:27:03.246999277Z")
}

func getDummyMessageWithLF(content string) *Message {
	return NewMessage([]byte(content), "info", len(content)+1, "2018-06-14T18:27:03.246999277Z")
}

func TestSingleLineHandler(t *testing.T) {
	outputChan := make(chan *Message, 10)
	h := NewSingleLineHandler(outputChan, 100)
	h.Start()

	var output *Message
	var line string

	// valid line should be sent
	line = "hello world"
	h.Handle(getDummyMessageWithLF(line))
	output = <-outputChan
	assert.Equal(t, line, string(output.Content))
	assert.Equal(t, len(line)+1, output.RawDataLen)

	// too long line should be truncated
	line = strings.Repeat("a", contentLenLimit+10)
	h.Handle(getDummyMessage(line))
	output = <-outputChan
	assert.Equal(t, len(line)+len(truncatedFlag), len(output.Content))
	assert.Equal(t, len(line), output.RawDataLen)

	line = strings.Repeat("a", contentLenLimit+10)
	h.Handle(getDummyMessage(line))
	output = <-outputChan
	assert.Equal(t, len(truncatedFlag)+len(line)+len(truncatedFlag), len(output.Content))
	assert.Equal(t, len(line), output.RawDataLen)

	line = strings.Repeat("a", 10)
	h.Handle(getDummyMessageWithLF(line))
	output = <-outputChan
	assert.Equal(t, string(truncatedFlag)+line, string(output.Content))
	assert.Equal(t, len(line)+1, output.RawDataLen)

	h.Stop()
}

func TestTrimSingleLine(t *testing.T) {
	outputChan := make(chan *Message, 10)
	h := NewSingleLineHandler(outputChan, 100)
	h.Start()

	var output *Message
	var line string

	// All leading and trailing whitespace characters should be trimmed
	line = whitespace + "foo" + whitespace + "bar" + whitespace
	h.Handle(getDummyMessageWithLF(line))
	output = <-outputChan
	assert.Equal(t, "foo"+whitespace+"bar", string(output.Content))
	assert.Equal(t, len(line)+1, output.RawDataLen)

	h.Stop()
}

func TestMultiLineHandler(t *testing.T) {
	re := regexp.MustCompile("[0-9]+\\.")
	outputChan := make(chan *Message, 10)
	h := NewMultiLineHandler(outputChan, re, 10*time.Millisecond, 20)
	h.Start()

	var output *Message

	// two lines long message should be sent
	h.Handle(getDummyMessageWithLF("1.first"))
	h.Handle(getDummyMessageWithLF("second"))

	// one line long message should be sent
	h.Handle(getDummyMessageWithLF("2. first line"))

	output = <-outputChan
	var expectedContent = "1.first\\nsecond"
	assert.Equal(t, expectedContent, string(output.Content))
	assert.Equal(t, len(expectedContent), output.RawDataLen)

	output = <-outputChan
	assert.Equal(t, "2. first line", string(output.Content))
	assert.Equal(t, len("2. first line")+1, output.RawDataLen)

	// too long line should be truncated
	h.Handle(getDummyMessage("3. stringssssssize20"))
	h.Handle(getDummyMessageWithLF("con"))

	output = <-outputChan
	assert.Equal(t, "3. stringssssssize20...TRUNCATED...", string(output.Content))
	assert.Equal(t, len("3. stringssssssize20"), output.RawDataLen)

	output = <-outputChan
	assert.Equal(t, "...TRUNCATED...con", string(output.Content))
	assert.Equal(t, 4, output.RawDataLen)

	// second line + TRUNCATED too long
	h.Handle(getDummyMessage("4. stringssssssize20"))
	h.Handle(getDummyMessageWithLF("continue"))

	output = <-outputChan
	assert.Equal(t, "4. stringssssssize20...TRUNCATED...", string(output.Content))
	assert.Equal(t, len("4. stringssssssize20"), output.RawDataLen)

	output = <-outputChan
	assert.Equal(t, "...TRUNCATED...continue...TRUNCATED...", string(output.Content))
	assert.Equal(t, 9, output.RawDataLen)

	// continuous too long lines
	h.Handle(getDummyMessage("5. stringssssssize20"))
	longLineTracingSpaces := "continu             "
	h.Handle(getDummyMessage(longLineTracingSpaces))
	h.Handle(getDummyMessageWithLF("end"))
	shortLineTracingSpaces := "6. next line      "
	h.Handle(getDummyMessageWithLF(shortLineTracingSpaces))

	output = <-outputChan
	assert.Equal(t, "5. stringssssssize20...TRUNCATED...", string(output.Content))
	assert.Equal(t, len("5. stringssssssize20"), output.RawDataLen)

	output = <-outputChan
	assert.Equal(t, "...TRUNCATED...continu             ...TRUNCATED...", string(output.Content))
	assert.Equal(t, len(longLineTracingSpaces), output.RawDataLen)

	output = <-outputChan
	assert.Equal(t, "...TRUNCATED...end", string(output.Content))
	assert.Equal(t, len("end\n"), output.RawDataLen)

	output = <-outputChan
	assert.Equal(t, "6. next line", string(output.Content))
	assert.Equal(t, len(shortLineTracingSpaces)+1, output.RawDataLen)

	h.Stop()
}

func TestTrimMultiLine(t *testing.T) {
	re := regexp.MustCompile("[0-9]+\\.")
	outputChan := make(chan *Message, 10)
	h := NewMultiLineHandler(outputChan, re, 10*time.Millisecond, 100)
	h.Start()

	var output *Message

	// All leading and trailing whitespace characters should be trimmed
	h.Handle(getDummyMessageWithLF(whitespace + "foo" + whitespace + "bar" + whitespace))
	output = <-outputChan
	assert.Equal(t, "foo"+whitespace+"bar", string(output.Content))
	assert.Equal(t, len(whitespace+"foo"+whitespace+"bar"+whitespace)+1, output.RawDataLen)

	// With line break
	h.Handle(getDummyMessageWithLF(whitespace + "foo" + whitespace))
	h.Handle(getDummyMessageWithLF("bar" + whitespace))
	output = <-outputChan
	assert.Equal(t, "foo"+whitespace+"\\n"+"bar", string(output.Content))
	assert.Equal(t, len(whitespace+"foo"+whitespace)+1+len("bar"+whitespace)+1, output.RawDataLen)

	h.Stop()
}

func TestMultiLineHandlerDropsEmptyMessages(t *testing.T) {
	outputChan := make(chan *Message, 10)
	re := regexp.MustCompile("[0-9]+\\.")
	h := NewMultiLineHandler(outputChan, re, 10*time.Millisecond, 100)
	h.Start()

	h.Handle(getDummyMessage(""))

	h.Handle(getDummyMessage("1.third line"))
	h.Handle(getDummyMessage("fourth line"))

	var output *Message

	output = <-outputChan
	assert.Equal(t, "1.third line\\nfourth line", string(output.Content))

	h.Stop()
}

func TestSingleLineHandlerSendsRawInvalidMessages(t *testing.T) {
	outputChan := make(chan *Message, 10)
	h := NewSingleLineHandler(outputChan, 100)
	h.Start()

	h.Handle(getDummyMessage("one message"))

	var output *Message

	output = <-outputChan
	assert.Equal(t, "one message", string(output.Content))
}

func TestMultiLineHandlerSendsRawInvalidMessages(t *testing.T) {
	outputChan := make(chan *Message, 10)
	re := regexp.MustCompile("[0-9]+\\.")
	h := NewMultiLineHandler(outputChan, re, 10*time.Millisecond, 100)
	h.Start()

	h.Handle(getDummyMessage("1.third line"))
	h.Handle(getDummyMessage("fourth line"))

	var output *Message

	output = <-outputChan
	assert.Equal(t, "1.third line\\nfourth line", string(output.Content))
}

func TestAutoMultiLineHandlerStaysSingleLineMode(t *testing.T) {

	outputChan := make(chan *Message, 10)
	source := config.NewLogSource("config", &config.LogsConfig{})
	h := NewAutoMultilineHandler(outputChan, 100, 5, 1.0, 10*time.Millisecond, 10*time.Millisecond, source)
	h.Start()

	for i := 0; i < 6; i++ {
		h.Handle(getDummyMessageWithLF("blah"))
		<-outputChan
	}
	assert.NotNil(t, h.singleLineHandler)
	assert.Nil(t, h.multiLineHandler)
	assert.Nil(t, source.GetPattern())
}

func TestAutoMultiLineHandlerSwitchesToMultiLineMode(t *testing.T) {

	outputChan := make(chan *Message, 10)
	source := config.NewLogSource("config", &config.LogsConfig{})
	h := NewAutoMultilineHandler(outputChan, 100, 5, 1.0, 10*time.Millisecond, 10*time.Millisecond, source)
	h.Start()

	for i := 0; i < 6; i++ {
		h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message"))
		<-outputChan
	}
	assert.Nil(t, h.singleLineHandler)
	assert.NotNil(t, h.multiLineHandler)
	assert.NotNil(t, source.GetPattern())
}

func TestAutoMultiLineHandlerHandelsMessage(t *testing.T) {

	outputChan := make(chan *Message, 10)
	source := config.NewLogSource("config", &config.LogsConfig{})
	h := NewAutoMultilineHandler(outputChan, 500, 1, 1.0, 10*time.Millisecond, 10*time.Millisecond, source)
	h.Start()

	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 1"))
	<-outputChan
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 2"))
	h.Handle(getDummyMessageWithLF("java.lang.Exception: boom"))
	h.Handle(getDummyMessageWithLF("at Main.funcd(Main.java:62)"))
	h.Handle(getDummyMessageWithLF("at Main.funcc(Main.java:60)"))
	h.Handle(getDummyMessageWithLF("at Main.funcb(Main.java:58)"))
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM another test message"))
	output := <-outputChan

	assert.Equal(t, "Jul 12, 2021 12:55:15 PM test message 2\\njava.lang.Exception: boom\\nat Main.funcd(Main.java:62)\\nat Main.funcc(Main.java:60)\\nat Main.funcb(Main.java:58)", string(output.Content))
}

func TestAutoMultiLineHandlerHandelsMessageConflictingPatterns(t *testing.T) {

	outputChan := make(chan *Message, 10)
	source := config.NewLogSource("config", &config.LogsConfig{})
	h := NewAutoMultilineHandler(outputChan, 500, 4, 0.75, 10*time.Millisecond, 10*time.Millisecond, source)
	h.Start()

	// we will match both patterns, but one will win with a threshold of 0.75
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 1"))
	h.Handle(getDummyMessageWithLF("Jul, 1-sep-12 10:20:30 pm test message 2"))
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 3"))
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 4"))

	for i := 0; i < 4; i++ {
		<-outputChan
	}
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 2"))
	h.Handle(getDummyMessageWithLF("java.lang.Exception: boom"))
	h.Handle(getDummyMessageWithLF("at Main.funcd(Main.java:62)"))
	h.Handle(getDummyMessageWithLF("at Main.funcc(Main.java:60)"))
	h.Handle(getDummyMessageWithLF("at Main.funcb(Main.java:58)"))
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM another test message"))
	output := <-outputChan

	assert.Equal(t, "Jul 12, 2021 12:55:15 PM test message 2\\njava.lang.Exception: boom\\nat Main.funcd(Main.java:62)\\nat Main.funcc(Main.java:60)\\nat Main.funcb(Main.java:58)", string(output.Content))
}

func TestAutoMultiLineHandlerHandelsMessageConflictingPatternsNoWinner(t *testing.T) {

	outputChan := make(chan *Message, 10)
	source := config.NewLogSource("config", &config.LogsConfig{})
	h := NewAutoMultilineHandler(outputChan, 500, 4, 0.75, 10*time.Millisecond, 10*time.Millisecond, source)
	h.Start()

	// we will match both patterns, but neither will win because it doesn't meet the threshold
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 1"))
	h.Handle(getDummyMessageWithLF("Jul, 1-sep-12 10:20:30 pm test message 2"))
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 3"))
	h.Handle(getDummyMessageWithLF("Jul, 1-sep-12 10:20:30 pm test message 4"))

	for i := 0; i < 4; i++ {
		<-outputChan
	}
	h.Handle(getDummyMessageWithLF("Jul 12, 2021 12:55:15 PM test message 2"))
	output := <-outputChan

	assert.NotNil(t, h.singleLineHandler)
	assert.Nil(t, h.multiLineHandler)

	assert.Equal(t, "Jul 12, 2021 12:55:15 PM test message 2", string(output.Content))
}
