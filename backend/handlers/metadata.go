package handlers

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func ExtractEPUBMetadata(epubPath, coverDir, fallbackTitle string) (title, author, coverPath string) {
	title = fallbackTitle

	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return
	}
	defer reader.Close()

	var container struct {
		Rootfiles struct {
			Rootfile []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfile"`
		} `xml:"rootfiles"`
	}
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
	var pkg struct {
		Metadata struct {
			Title   []string   `xml:"title"`
			Creator []string   `xml:"creator"`
			Meta    []epubMeta `xml:"meta"`
		} `xml:"metadata"`
		Manifest struct {
			Items []epubItem `xml:"item"`
		} `xml:"manifest"`
	}
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

	buffer := make([]byte, 50000)
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
	ext := ".jpg"
	if isPNG(data) {
		ext = ".png"
	}
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return ""
	}
	coverPath := filepath.Join(coverDir, "cover"+ext)
	if err := os.WriteFile(coverPath, data, 0644); err != nil {
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
	if end == -1 || end >= 200 {
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
