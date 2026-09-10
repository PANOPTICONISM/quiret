package handlers

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// --- minimal MP4/box builders for tests ---

func box(typ string, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(b[0:4], uint32(8+len(payload)))
	copy(b[4:8], typ)
	copy(b[8:], payload)
	return b
}

func u16(v uint16) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, v); return b }
func u32(v uint32) []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, v); return b }
func u64(v uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, v); return b }

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func hdlr(handlerType string) []byte {
	return box("hdlr", concat([]byte{0, 0, 0, 0}, []byte{0, 0, 0, 0}, []byte(handlerType), []byte{0, 0, 0, 0}))
}

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return path
}

// --- parseChplAtom ---

func TestParseChplAtom(t *testing.T) {
	payload := concat(
		[]byte{1, 0, 0, 0}, // version 1 + flags
		[]byte{0, 0, 0, 0}, // reserved (version != 0)
		[]byte{2},          // chapter count
		u64(0), []byte{byte(len("Intro"))}, []byte("Intro"),
		u64(900000000), []byte{byte(len("Chapter One"))}, []byte("Chapter One"), // 90s in 100ns units
	)

	chapters := parseChplAtom(payload)
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "Intro" || chapters[0].Start != 0 {
		t.Errorf("chapter 0 = %+v", chapters[0])
	}
	if chapters[1].Title != "Chapter One" || chapters[1].Start != 90 {
		t.Errorf("chapter 1 = %+v", chapters[1])
	}
}

// --- ExtractAudioChapters: Nero chpl ---

func TestExtractAudioChaptersNero(t *testing.T) {
	chpl := box("chpl", concat(
		[]byte{1, 0, 0, 0}, []byte{0, 0, 0, 0}, []byte{2},
		u64(0), []byte{byte(len("One"))}, []byte("One"),
		u64(600000000), []byte{byte(len("Two"))}, []byte("Two"), // 60s
	))
	file := concat(
		box("ftyp", []byte("M4A mp42")),
		box("moov", box("udta", chpl)),
	)
	path := writeTemp(t, "nero.m4b", file)

	chapters := ExtractAudioChapters(path)
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "One" || chapters[0].Start != 0 {
		t.Errorf("chapter 0 = %+v", chapters[0])
	}
	if chapters[1].Title != "Two" || chapters[1].Start != 60 {
		t.Errorf("chapter 1 = %+v", chapters[1])
	}
}

// --- ExtractAudioChapters: QuickTime text track ---

func TestExtractAudioChaptersTextTrack(t *testing.T) {
	// Chapter text samples: 2-byte length prefix + UTF-8 title.
	sample0 := concat(u16(uint16(len("Intro"))), []byte("Intro"))
	sample1 := concat(u16(uint16(len("Chapter 1"))), []byte("Chapter 1"))
	mdatPayload := concat(sample0, sample1)

	ftyp := box("ftyp", []byte("M4A mp42"))
	mdat := box("mdat", mdatPayload)
	sampleBase := uint32(len(ftyp) + 8) // start of mdat payload

	// Chapter track sample tables (timescale 1000; durations 90s then 30s).
	stts := box("stts", concat([]byte{0, 0, 0, 0}, u32(2), u32(1), u32(90000), u32(1), u32(30000)))
	stsz := box("stsz", concat([]byte{0, 0, 0, 0}, u32(0), u32(2), u32(uint32(len(sample0))), u32(uint32(len(sample1)))))
	stsc := box("stsc", concat([]byte{0, 0, 0, 0}, u32(1), u32(1), u32(2), u32(1)))
	stco := box("stco", concat([]byte{0, 0, 0, 0}, u32(1), u32(sampleBase)))
	stbl := box("stbl", concat(stts, stsz, stsc, stco))
	minf := box("minf", stbl)
	mdhd := box("mdhd", concat([]byte{0, 0, 0, 0}, u32(0), u32(0), u32(1000), u32(120000)))
	tkhdChap := box("tkhd", concat([]byte{0, 0, 0, 0}, u32(0), u32(0), u32(2)))
	chapterTrak := box("trak", concat(tkhdChap, box("mdia", concat(mdhd, hdlr("text"), minf))))

	tkhdAudio := box("tkhd", concat([]byte{0, 0, 0, 0}, u32(0), u32(0), u32(1)))
	tref := box("tref", box("chap", u32(2))) // audio track references chapter track id 2
	audioTrak := box("trak", concat(tkhdAudio, tref, box("mdia", hdlr("soun"))))

	moov := box("moov", concat(audioTrak, chapterTrak))
	file := concat(ftyp, mdat, moov)
	path := writeTemp(t, "texttrack.m4b", file)

	chapters := ExtractAudioChapters(path)
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d: %+v", len(chapters), chapters)
	}
	if chapters[0].Title != "Intro" || chapters[0].Start != 0 {
		t.Errorf("chapter 0 = %+v", chapters[0])
	}
	if chapters[1].Title != "Chapter 1" || chapters[1].Start != 90 {
		t.Errorf("chapter 1 = %+v", chapters[1])
	}
}

func TestExtractAudioChaptersNone(t *testing.T) {
	file := concat(
		box("ftyp", []byte("M4A mp42")),
		box("moov", box("mvhd", make([]byte, 100))),
	)
	path := writeTemp(t, "none.m4b", file)
	if chapters := ExtractAudioChapters(path); chapters != nil {
		t.Errorf("expected nil, got %+v", chapters)
	}
}

func TestExtractAudioChaptersWrongExtension(t *testing.T) {
	path := writeTemp(t, "book.mp3", []byte("id3 or whatever"))
	if chapters := ExtractAudioChapters(path); chapters != nil {
		t.Errorf("mp3 should have no MP4 chapters, got %+v", chapters)
	}
}

// --- ExtractAudioMetadata fallback ---

func TestExtractAudioMetadataFallback(t *testing.T) {
	path := writeTemp(t, "untagged.mp3", []byte("this is not a real audio file"))
	title, author, cover := ExtractAudioMetadata(path, t.TempDir(), "My Fallback Title")
	if title != "My Fallback Title" {
		t.Errorf("title = %q, want fallback", title)
	}
	if author != "" {
		t.Errorf("author = %q, want empty", author)
	}
	if cover != "" {
		t.Errorf("cover = %q, want empty", cover)
	}
}

// --- IsImageFile ---

func TestIsImageFile(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0}
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	notImage := []byte("plain text content")

	if !IsImageFile(jpeg) {
		t.Error("JPEG magic not recognized")
	}
	if !IsImageFile(png) {
		t.Error("PNG magic not recognized")
	}
	if IsImageFile(notImage) {
		t.Error("text wrongly recognized as image")
	}
}
