package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"quiret/models"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/dhowden/tag"
	"golang.org/x/image/draw"
)

const (
	coverMaxWidth     = 320
	coverMaxDecodeDim = 8000
	pdfHeaderBytes    = 50000
	pdfFieldMaxLen    = 200
)

type epubContainer struct {
	Rootfiles struct {
		Rootfile []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type epubItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type epubMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

type epubPackage struct {
	Metadata struct {
		Title   []string   `xml:"title"`
		Creator []string   `xml:"creator"`
		Meta    []epubMeta `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []epubItem `xml:"item"`
	} `xml:"manifest"`
}

type fb2Image struct {
	Href string `xml:"href,attr"`
}

type fb2Author struct {
	FirstName  string `xml:"first-name"`
	MiddleName string `xml:"middle-name"`
	LastName   string `xml:"last-name"`
	Nickname   string `xml:"nickname"`
}

type fb2Coverpage struct {
	Images []fb2Image `xml:"image"`
}

type fb2TitleInfo struct {
	BookTitle string       `xml:"book-title"`
	Authors   []fb2Author  `xml:"author"`
	Coverpage fb2Coverpage `xml:"coverpage"`
}

type fb2Description struct {
	TitleInfo fb2TitleInfo `xml:"title-info"`
}

type fb2Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"`
}

type fb2Book struct {
	XMLName     xml.Name       `xml:"FictionBook"`
	Description fb2Description `xml:"description"`
	Binaries    []fb2Binary    `xml:"binary"`
}

func ExtractEPUBMetadata(epubPath, coverDir, fallbackTitle string) (title, author, coverPath string) {
	title = fallbackTitle

	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return
	}
	defer reader.Close()

	var container epubContainer
	containerData, err := readZipFile(reader, "META-INF/container.xml")
	if err != nil {
		return
	}
	if err := xml.Unmarshal(containerData, &container); err != nil {
		return
	}
	if len(container.Rootfiles.Rootfile) == 0 {
		return
	}
	opfPath := container.Rootfiles.Rootfile[0].FullPath
	if opfPath == "" {
		return
	}

	var pkg epubPackage
	opfData, err := readZipFile(reader, opfPath)
	if err != nil {
		return
	}
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return
	}

	if len(pkg.Metadata.Title) > 0 {
		if t := strings.TrimSpace(pkg.Metadata.Title[0]); t != "" {
			title = t
		}
	}
	if len(pkg.Metadata.Creator) > 0 {
		author = strings.TrimSpace(pkg.Metadata.Creator[0])
	}

	var coverHref string
	opfDir := filepath.Dir(opfPath)

	for _, it := range pkg.Manifest.Items {
		for _, p := range strings.Fields(it.Properties) {
			if p == "cover-image" {
				coverHref = it.Href
				break
			}
		}
		if coverHref != "" {
			break
		}
	}

	if coverHref == "" {
		var coverID string
		for _, m := range pkg.Metadata.Meta {
			if m.Name == "cover" {
				coverID = m.Content
				break
			}
		}
		if coverID != "" {
			for _, it := range pkg.Manifest.Items {
				if it.ID == coverID && strings.HasPrefix(it.MediaType, "image/") {
					coverHref = it.Href
					break
				}
			}
		}
	}

	if coverHref == "" {
		commonCovers := map[string]bool{"cover.jpg": true, "cover.jpeg": true, "cover.png": true}
		for _, f := range reader.File {
			if commonCovers[strings.ToLower(filepath.Base(f.Name))] {
				coverHref = f.Name
				opfDir = ""
				break
			}
		}
	}

	if coverHref == "" {
		return
	}

	if strings.Contains(coverHref, "..") {
		return
	}

	var coverInZip string
	if opfDir != "" && opfDir != "." {
		coverInZip = filepath.Join(opfDir, coverHref)
	} else {
		coverInZip = coverHref
	}
	coverInZip = filepath.ToSlash(coverInZip)
	coverInZip = strings.ReplaceAll(coverInZip, "%20", " ")

	for _, f := range reader.File {
		if filepath.ToSlash(f.Name) != coverInZip {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			break
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			break
		}
		coverPath = writeCoverImage(coverDir, data)
		if coverPath == "" {
			log.Printf("Cover file %s could not be saved as a valid image", f.Name)
		}
		break
	}

	return
}

func readZipFile(reader *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range reader.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found in zip: %s", name)
}

func ExtractPDFMetadata(pdfPath, originalName string) (title, author string) {
	title = originalName
	author = ""

	file, err := os.Open(pdfPath)
	if err != nil {
		return
	}
	defer file.Close()

	buffer := make([]byte, pdfHeaderBytes)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return
	}
	content := string(buffer[:n])

	if t := extractPDFField(content, "/Title"); len(t) > 2 {
		title = t
	}
	if a := extractPDFField(content, "/Author"); a != "" {
		author = a
	}

	return
}

func ExtractPDFCover(pdfPath, coverDir, bookID string) string {
	tempBase := filepath.Join(os.TempDir(), "quiret-pdf-"+bookID)
	tempCover := tempBase + "-001.jpg"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx,
		"pdftoppm",
		"-jpeg",
		"-f", "1",
		"-l", "1",
		"-scale-to", "800",
		pdfPath,
		tempBase,
	)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("PDF cover extraction timed out for: %s", pdfPath)
		} else {
			log.Printf("Failed to extract PDF cover with pdftoppm: %v", err)
		}
		return ""
	}

	if _, err := os.Stat(tempCover); err != nil {
		return ""
	}
	defer os.Remove(tempCover)

	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return ""
	}

	finalPath := filepath.Join(coverDir, "cover.jpg")
	if err := copyFile(tempCover, finalPath); err != nil {
		return ""
	}

	return finalPath
}

func ExtractFB2Metadata(fb2Path, coverDir, fallbackTitle string) (title, author, coverPath string) {
	title = fallbackTitle

	data, err := os.ReadFile(fb2Path)
	if err != nil {
		return
	}

	var book fb2Book
	if err := xml.Unmarshal(data, &book); err != nil {
		return
	}

	if t := strings.TrimSpace(book.Description.TitleInfo.BookTitle); t != "" {
		title = t
	}

	if len(book.Description.TitleInfo.Authors) > 0 {
		a := book.Description.TitleInfo.Authors[0]
		var parts []string
		for _, p := range []string{a.FirstName, a.MiddleName, a.LastName} {
			if s := strings.TrimSpace(p); s != "" {
				parts = append(parts, s)
			}
		}
		author = strings.Join(parts, " ")
		if author == "" {
			author = strings.TrimSpace(a.Nickname)
		}
	}

	var coverID string
	for _, img := range book.Description.TitleInfo.Coverpage.Images {
		if href := strings.TrimPrefix(img.Href, "#"); href != "" {
			coverID = href
			break
		}
	}
	if coverID == "" {
		return
	}

	for _, b := range book.Binaries {
		if b.ID != coverID {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(b.Data), ""))
		if err != nil {
			break
		}
		coverPath = writeCoverImage(coverDir, raw)
		break
	}

	return
}

func ExtractCBZCover(cbzPath, coverDir string) string {
	reader, err := zip.OpenReader(cbzPath)
	if err != nil {
		return ""
	}
	defer reader.Close()

	var imageFiles []*zip.File
	for _, f := range reader.File {
		name := strings.ToLower(f.Name)
		if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") ||
			strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".webp") {
			imageFiles = append(imageFiles, f)
		}
	}

	if len(imageFiles) == 0 {
		return ""
	}

	sort.Slice(imageFiles, func(i, j int) bool {
		return imageFiles[i].Name < imageFiles[j].Name
	})

	rc, err := imageFiles[0].Open()
	if err != nil {
		return ""
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}

	return writeCoverImage(coverDir, data)
}

// ExtractAudioMetadata reads title, author and embedded cover art from an audio
// file (MP3, M4B, M4A, AAC, OGG, ...) using the file's tags. Falls back to the
// provided title when tags are missing.
func ExtractAudioMetadata(audioPath, coverDir, fallbackTitle string) (title, author, coverPath string) {
	title = fallbackTitle

	file, err := os.Open(audioPath)
	if err != nil {
		return
	}
	defer file.Close()

	m, err := tag.ReadFrom(file)
	if err != nil {
		// Not fatal: many valid audio files carry no readable tags.
		return
	}

	if t := strings.TrimSpace(m.Title()); t != "" {
		title = t
	} else if al := strings.TrimSpace(m.Album()); al != "" {
		title = al
	}

	author = strings.TrimSpace(m.Artist())
	if author == "" {
		author = strings.TrimSpace(m.AlbumArtist())
	}

	if pic := m.Picture(); pic != nil && len(pic.Data) > 0 {
		coverPath = writeCoverImage(coverDir, pic.Data)
	}

	return
}

// ExtractAudioChapters returns embedded chapter markers for MP4-based audiobooks
// (.m4b/.m4a). It first tries the Nero-style "chpl" atom (moov > udta > chpl), then
// falls back to a QuickTime/MP4 chapter text track. Returns nil for formats without
// embedded chapters or when none are found.
func ExtractAudioChapters(audioPath string) []models.Chapter {
	ext := strings.ToLower(filepath.Ext(audioPath))
	if ext != ".m4b" && ext != ".m4a" && ext != ".mp4" {
		return nil
	}

	file, err := os.Open(audioPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil
	}
	size := info.Size()

	moovStart, moovEnd, ok := findMP4Atom(file, 0, size, "moov")
	if !ok {
		return nil
	}

	if ch := neroChapters(file, moovStart, moovEnd); len(ch) > 0 {
		return ch
	}
	if ch := textTrackChapters(file, moovStart, moovEnd); len(ch) > 0 {
		return ch
	}
	return nil
}

// neroChapters reads a Nero-style "chpl" chapter list (moov > udta > chpl).
func neroChapters(r io.ReaderAt, moovStart, moovEnd int64) []models.Chapter {
	cs, ce, ok := findMP4Path(r, moovStart, moovEnd, "udta", "chpl")
	if !ok {
		return nil
	}
	buf, ok := readBox(r, cs, ce, 1<<20)
	if !ok {
		return nil
	}
	return parseChplAtom(buf)
}

type mp4Box struct {
	typ        string
	start, end int64 // payload range
}

// mp4Boxes lists the child boxes directly within [start, end).
func mp4Boxes(r io.ReaderAt, start, end int64) []mp4Box {
	var boxes []mp4Box
	pos := start
	header := make([]byte, 8)
	for pos+8 <= end {
		if _, err := r.ReadAt(header, pos); err != nil {
			break
		}
		boxSize := int64(binary.BigEndian.Uint32(header[0:4]))
		boxType := string(header[4:8])

		var payloadStart, payloadEnd int64
		switch boxSize {
		case 1: // 64-bit extended size follows the header
			ext := make([]byte, 8)
			if _, err := r.ReadAt(ext, pos+8); err != nil {
				return boxes
			}
			boxSize = int64(binary.BigEndian.Uint64(ext))
			payloadStart = pos + 16
			payloadEnd = pos + boxSize
		case 0: // box extends to the end of the container
			payloadStart = pos + 8
			payloadEnd = end
		default:
			payloadStart = pos + 8
			payloadEnd = pos + boxSize
		}

		if boxSize != 0 && boxSize < 8 {
			break // malformed box
		}
		if payloadEnd <= pos || payloadEnd > end {
			break
		}
		boxes = append(boxes, mp4Box{typ: boxType, start: payloadStart, end: payloadEnd})
		if len(boxes) > 10000 { // guard against pathological files
			break
		}
		pos = payloadEnd
	}
	return boxes
}

// findMP4Atom returns the payload range of the first child box matching want.
func findMP4Atom(r io.ReaderAt, start, end int64, want string) (int64, int64, bool) {
	for _, b := range mp4Boxes(r, start, end) {
		if b.typ == want {
			return b.start, b.end, true
		}
	}
	return 0, 0, false
}

// findMP4Path walks a nested chain of box types, e.g. "mdia","minf","stbl".
func findMP4Path(r io.ReaderAt, start, end int64, path ...string) (int64, int64, bool) {
	s, e := start, end
	for _, t := range path {
		ns, ne, ok := findMP4Atom(r, s, e, t)
		if !ok {
			return 0, 0, false
		}
		s, e = ns, ne
	}
	return s, e, true
}

// readBox reads a box payload into memory, rejecting anything larger than max.
func readBox(r io.ReaderAt, start, end, max int64) ([]byte, bool) {
	n := end - start
	if n <= 0 || n > max {
		return nil, false
	}
	buf := make([]byte, n)
	if _, err := r.ReadAt(buf, start); err != nil {
		return nil, false
	}
	return buf, true
}

// textTrackChapters extracts chapters from a QuickTime/MP4 chapter text track: a
// text/tx3g track referenced by another track's "chap" track-reference (or, failing
// that, the first text/subtitle track). Chapter start times come from the track's
// sample durations (stts) and titles from the length-prefixed text samples located
// via the sample tables (stsc/stsz/stco).
func textTrackChapters(r io.ReaderAt, moovStart, moovEnd int64) []models.Chapter {
	var traks []mp4Box
	for _, b := range mp4Boxes(r, moovStart, moovEnd) {
		if b.typ == "trak" {
			traks = append(traks, b)
		}
	}
	if len(traks) == 0 {
		return nil
	}

	trackByID := make(map[uint32]mp4Box)
	var chapRefID uint32
	var textTrak *mp4Box
	for i := range traks {
		t := traks[i]
		if id := trakID(r, t); id != 0 {
			trackByID[id] = t
		}
		if ts, te, ok := findMP4Path(r, t.start, t.end, "tref", "chap"); ok {
			if buf, ok := readBox(r, ts, te, 4096); ok && len(buf) >= 4 {
				chapRefID = binary.BigEndian.Uint32(buf[:4])
			}
		}
		if textTrak == nil {
			if h := trakHandler(r, t); h == "text" || h == "sbtl" {
				tt := t
				textTrak = &tt
			}
		}
	}

	var chapTrak mp4Box
	switch {
	case chapRefID != 0 && trackByID[chapRefID].end != 0:
		chapTrak = trackByID[chapRefID]
	case textTrak != nil:
		chapTrak = *textTrak
	default:
		return nil
	}

	ms, me, ok := findMP4Path(r, chapTrak.start, chapTrak.end, "mdia", "mdhd")
	if !ok {
		return nil
	}
	mdhd, ok := readBox(r, ms, me, 64)
	if !ok {
		return nil
	}
	timescale := mdhdTimescale(mdhd)
	if timescale == 0 {
		return nil
	}

	ss, se, ok := findMP4Path(r, chapTrak.start, chapTrak.end, "mdia", "minf", "stbl")
	if !ok {
		return nil
	}

	durations := parseStts(r, ss, se)
	sizes := parseStsz(r, ss, se)
	offsets := parseSampleOffsets(r, ss, se, sizes)
	n := min(len(durations), len(sizes), len(offsets))
	if n == 0 {
		return nil
	}
	if n > 5000 {
		n = 5000
	}

	chapters := make([]models.Chapter, 0, n)
	var cumulative uint64
	for i := 0; i < n; i++ {
		start := float64(cumulative) / float64(timescale)
		cumulative += uint64(durations[i])
		title := readTextSample(r, offsets[i], sizes[i])
		if title == "" {
			title = fmt.Sprintf("Chapter %d", i+1)
		}
		chapters = append(chapters, models.Chapter{Title: title, Start: start})
	}
	return chapters
}

// trakID returns a track's ID from its tkhd box (0 if unavailable).
func trakID(r io.ReaderAt, trak mp4Box) uint32 {
	ts, te, ok := findMP4Atom(r, trak.start, trak.end, "tkhd")
	if !ok {
		return 0
	}
	buf, ok := readBox(r, ts, te, 256)
	if !ok || len(buf) < 4 {
		return 0
	}
	if buf[0] == 1 { // 64-bit creation/modification times
		if len(buf) < 24 {
			return 0
		}
		return binary.BigEndian.Uint32(buf[20:24])
	}
	if len(buf) < 16 {
		return 0
	}
	return binary.BigEndian.Uint32(buf[12:16])
}

// trakHandler returns a track's handler type (e.g. "soun", "text", "sbtl").
func trakHandler(r io.ReaderAt, trak mp4Box) string {
	hs, he, ok := findMP4Path(r, trak.start, trak.end, "mdia", "hdlr")
	if !ok {
		return ""
	}
	buf, ok := readBox(r, hs, he, 256)
	if !ok || len(buf) < 12 {
		return ""
	}
	return string(buf[8:12])
}

func mdhdTimescale(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	if b[0] == 1 {
		if len(b) < 24 {
			return 0
		}
		return binary.BigEndian.Uint32(b[20:24])
	}
	if len(b) < 16 {
		return 0
	}
	return binary.BigEndian.Uint32(b[12:16])
}

// parseStts expands the time-to-sample table into per-sample durations.
func parseStts(r io.ReaderAt, stblStart, stblEnd int64) []uint32 {
	bs, be, ok := findMP4Atom(r, stblStart, stblEnd, "stts")
	if !ok {
		return nil
	}
	buf, ok := readBox(r, bs, be, 1<<20)
	if !ok || len(buf) < 8 {
		return nil
	}
	entryCount := binary.BigEndian.Uint32(buf[4:8])
	p := 8
	var out []uint32
	for i := uint32(0); i < entryCount; i++ {
		if p+8 > len(buf) {
			break
		}
		count := binary.BigEndian.Uint32(buf[p : p+4])
		delta := binary.BigEndian.Uint32(buf[p+4 : p+8])
		p += 8
		if count > 100000 {
			count = 100000
		}
		for j := uint32(0); j < count; j++ {
			out = append(out, delta)
			if len(out) > 20000 {
				return out
			}
		}
	}
	return out
}

// parseStsz returns per-sample sizes from the sample-size table.
func parseStsz(r io.ReaderAt, stblStart, stblEnd int64) []uint32 {
	bs, be, ok := findMP4Atom(r, stblStart, stblEnd, "stsz")
	if !ok {
		return nil
	}
	buf, ok := readBox(r, bs, be, 1<<20)
	if !ok || len(buf) < 12 {
		return nil
	}
	sampleSize := binary.BigEndian.Uint32(buf[4:8])
	sampleCount := binary.BigEndian.Uint32(buf[8:12])
	if sampleCount > 20000 {
		sampleCount = 20000
	}
	out := make([]uint32, 0, sampleCount)
	if sampleSize != 0 {
		for i := uint32(0); i < sampleCount; i++ {
			out = append(out, sampleSize)
		}
		return out
	}
	p := 12
	for i := uint32(0); i < sampleCount; i++ {
		if p+4 > len(buf) {
			break
		}
		out = append(out, binary.BigEndian.Uint32(buf[p:p+4]))
		p += 4
	}
	return out
}

// parseSampleOffsets computes the absolute file offset of each sample using the
// sample-to-chunk (stsc) and chunk-offset (stco/co64) tables plus sample sizes.
func parseSampleOffsets(r io.ReaderAt, stblStart, stblEnd int64, sizes []uint32) []int64 {
	var chunkOffsets []int64
	if bs, be, ok := findMP4Atom(r, stblStart, stblEnd, "stco"); ok {
		if buf, ok := readBox(r, bs, be, 1<<20); ok && len(buf) >= 8 {
			count := binary.BigEndian.Uint32(buf[4:8])
			p := 8
			for i := uint32(0); i < count && p+4 <= len(buf); i++ {
				chunkOffsets = append(chunkOffsets, int64(binary.BigEndian.Uint32(buf[p:p+4])))
				p += 4
			}
		}
	} else if bs, be, ok := findMP4Atom(r, stblStart, stblEnd, "co64"); ok {
		if buf, ok := readBox(r, bs, be, 1<<20); ok && len(buf) >= 8 {
			count := binary.BigEndian.Uint32(buf[4:8])
			p := 8
			for i := uint32(0); i < count && p+8 <= len(buf); i++ {
				chunkOffsets = append(chunkOffsets, int64(binary.BigEndian.Uint64(buf[p:p+8])))
				p += 8
			}
		}
	}
	if len(chunkOffsets) == 0 {
		return nil
	}

	type stscRun struct{ firstChunk, samplesPerChunk uint32 }
	var runs []stscRun
	if bs, be, ok := findMP4Atom(r, stblStart, stblEnd, "stsc"); ok {
		if buf, ok := readBox(r, bs, be, 1<<20); ok && len(buf) >= 8 {
			count := binary.BigEndian.Uint32(buf[4:8])
			p := 8
			for i := uint32(0); i < count && p+12 <= len(buf); i++ {
				runs = append(runs, stscRun{
					firstChunk:      binary.BigEndian.Uint32(buf[p : p+4]),
					samplesPerChunk: binary.BigEndian.Uint32(buf[p+4 : p+8]),
				})
				p += 12
			}
		}
	}
	if len(runs) == 0 {
		return nil
	}

	samplesPerChunk := func(chunk uint32) uint32 {
		var spc uint32
		for _, rn := range runs {
			if rn.firstChunk <= chunk {
				spc = rn.samplesPerChunk
			} else {
				break
			}
		}
		return spc
	}

	offsets := make([]int64, 0, len(sizes))
	sampleIdx := 0
	for ci := 1; ci <= len(chunkOffsets); ci++ {
		spc := samplesPerChunk(uint32(ci))
		if spc > 20000 {
			spc = 20000
		}
		base := chunkOffsets[ci-1]
		var within int64
		for k := uint32(0); k < spc; k++ {
			if sampleIdx >= len(sizes) {
				return offsets
			}
			offsets = append(offsets, base+within)
			within += int64(sizes[sampleIdx])
			sampleIdx++
		}
	}
	return offsets
}

// readTextSample reads one chapter text sample: a 2-byte length prefix followed by
// the title (UTF-8, or UTF-16 with a byte-order mark).
func readTextSample(r io.ReaderAt, offset int64, size uint32) string {
	if size < 2 || size > 65536 {
		return ""
	}
	buf := make([]byte, size)
	if _, err := r.ReadAt(buf, offset); err != nil {
		return ""
	}
	textLen := int(binary.BigEndian.Uint16(buf[0:2]))
	if 2+textLen > len(buf) {
		textLen = len(buf) - 2
	}
	return strings.TrimSpace(decodeText(buf[2 : 2+textLen]))
}

func decodeText(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF { // UTF-16 BE BOM
		u := make([]uint16, 0, (len(b)-2)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
		}
		return string(utf16.Decode(u))
	}
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE { // UTF-16 LE BOM
		u := make([]uint16, 0, (len(b)-2)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u = append(u, uint16(b[i+1])<<8|uint16(b[i]))
		}
		return string(utf16.Decode(u))
	}
	return string(b)
}

// parseChplAtom decodes a Nero "chpl" chapter-list box payload. The layout follows
// the widely-used convention (see FFmpeg's mov_read_chpl): a full-box header, an
// optional 32-bit field for version != 0, a one-byte chapter count, then for each
// chapter an 8-byte start time (in 100-nanosecond units) and a length-prefixed title.
func parseChplAtom(b []byte) []models.Chapter {
	if len(b) < 5 {
		return nil
	}
	version := b[0]
	p := 4 // skip version (1) + flags (3)
	if version != 0 {
		if len(b) < p+4 {
			return nil
		}
		p += 4
	}
	if p >= len(b) {
		return nil
	}
	count := int(b[p])
	p++
	if count <= 0 || count > 5000 {
		return nil
	}

	chapters := make([]models.Chapter, 0, count)
	for i := 0; i < count; i++ {
		if p+9 > len(b) {
			break
		}
		start := binary.BigEndian.Uint64(b[p : p+8])
		p += 8
		titleLen := int(b[p])
		p++
		if p+titleLen > len(b) {
			break
		}
		title := strings.TrimSpace(string(b[p : p+titleLen]))
		p += titleLen
		chapters = append(chapters, models.Chapter{
			Title: title,
			Start: float64(start) / 1e7, // 100ns units -> seconds
		})
	}
	if len(chapters) == 0 {
		return nil
	}
	return chapters
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func writeCoverImage(coverDir string, data []byte) string {
	if !IsImageFile(data) {
		return ""
	}
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return ""
	}

	saveOriginal := func() string {
		ext := ".jpg"
		if isPNG(data) {
			ext = ".png"
		}
		coverPath := filepath.Join(coverDir, "cover"+ext)
		if writeErr := os.WriteFile(coverPath, data, 0644); writeErr != nil {
			return ""
		}
		return coverPath
	}

	const maxCoverDim = 8000
	cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(data))
	if cfgErr != nil || cfg.Width > maxCoverDim || cfg.Height > maxCoverDim {
		if cfgErr == nil {
			log.Printf("Cover too large to thumbnail (%dx%d), storing original",
				cfg.Width, cfg.Height)
		}
		return saveOriginal()
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return saveOriginal()
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	dstW, dstH := srcW, srcH
	if srcW > coverMaxWidth {
		dstW = coverMaxWidth
		dstH = srcH * coverMaxWidth / srcW
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	coverPath := filepath.Join(coverDir, "cover.jpg")
	out, err := os.Create(coverPath)
	if err != nil {
		return ""
	}
	defer out.Close()
	if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return ""
	}
	return coverPath
}

func isPNG(data []byte) bool {
	return len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50
}

func extractPDFField(content, key string) string {
	idx := strings.Index(content, key)
	if idx == -1 {
		return ""
	}
	start := strings.Index(content[idx:], "(")
	if start == -1 {
		return ""
	}
	start += idx + 1
	end := strings.Index(content[start:], ")")
	if end == -1 || end >= pdfFieldMaxLen {
		return ""
	}
	val := strings.TrimSpace(content[start : start+end])
	val = strings.ReplaceAll(val, "\\(", "(")
	val = strings.ReplaceAll(val, "\\)", ")")
	return val
}

// IsImageFile checks if data represents a valid image file by checking magic bytes.
func IsImageFile(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true
	}
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}
	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return true
	}
	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) > 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		if data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
			return true
		}
	}
	return false
}
