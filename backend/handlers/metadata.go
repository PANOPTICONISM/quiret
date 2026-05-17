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

// ExtractEPUBMetadata extracts title, author, and cover from an EPUB file.
// coverDir is where the cover image will be saved (as "cover.jpg" or "cover.png").
func ExtractEPUBMetadata(epubPath, coverDir, fallbackTitle string) (title, author, coverPath string) {
	title = fallbackTitle
	author = ""
	coverPath = ""

	reader, err := zip.OpenReader(epubPath)
	if err != nil {
		return
	}
	defer reader.Close()

	var containerXML string
	var opfPath string

	for _, f := range reader.File {
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			containerXML = string(data)
			break
		}
	}

	if idx := strings.Index(containerXML, "full-path=\""); idx != -1 {
		start := idx + 11
		end := strings.Index(containerXML[start:], "\"")
		if end != -1 {
			opfPath = containerXML[start : start+end]
		}
	}

	if opfPath == "" {
		return
	}

	var opfXML string
	for _, f := range reader.File {
		if f.Name == opfPath {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			opfXML = string(data)
			break
		}
	}

	if titleIdx := strings.Index(opfXML, "<dc:title>"); titleIdx != -1 {
		start := titleIdx + 10
		end := strings.Index(opfXML[start:], "</dc:title>")
		if end != -1 {
			title = opfXML[start : start+end]
		}
	}

	if authorIdx := strings.Index(opfXML, "<dc:creator"); authorIdx != -1 {
		contentStart := strings.Index(opfXML[authorIdx:], ">")
		if contentStart != -1 {
			start := authorIdx + contentStart + 1
			end := strings.Index(opfXML[start:], "</dc:creator>")
			if end != -1 {
				author = opfXML[start : start+end]
			}
		}
	}

	opfDir := filepath.Dir(opfPath)
	var coverHref string

	// Strategy 1: EPUB 3 - Look for item with properties="cover-image"
	if idx := strings.Index(opfXML, "properties=\"cover-image\""); idx != -1 {
		itemStart := strings.LastIndex(opfXML[:idx], "<item")
		if itemStart != -1 {
			itemEnd := strings.Index(opfXML[itemStart:], "/>")
			if itemEnd == -1 {
				itemEnd = strings.Index(opfXML[itemStart:], "</item>")
			}
			if itemEnd != -1 {
				itemTag := opfXML[itemStart : itemStart+itemEnd]
				if hrefIdx := strings.Index(itemTag, "href=\""); hrefIdx != -1 {
					start := hrefIdx + 6
					end := strings.Index(itemTag[start:], "\"")
					if end != -1 {
						coverHref = itemTag[start : start+end]
					}
				}
			}
		}
	}

	// Strategy 2: EPUB 2 - Look for meta name="cover" content="cover-id"
	if coverHref == "" {
		var coverID string
		if coverIdx := strings.Index(opfXML, "name=\"cover\""); coverIdx != -1 {
			metaStart := strings.LastIndex(opfXML[:coverIdx], "<meta")
			if metaStart != -1 {
				metaEnd := strings.Index(opfXML[metaStart:], "/>")
				if metaEnd == -1 {
					metaEnd = strings.Index(opfXML[metaStart:], ">")
				}
				if metaEnd != -1 {
					metaTag := opfXML[metaStart : metaStart+metaEnd]
					if contentIdx := strings.Index(metaTag, "content=\""); contentIdx != -1 {
						start := contentIdx + 9
						end := strings.Index(metaTag[start:], "\"")
						if end != -1 {
							coverID = metaTag[start : start+end]
						}
					}
				}
			}
		}

		if coverID != "" {
			searchStr := fmt.Sprintf("id=\"%s\"", coverID)
			if itemIdx := strings.Index(opfXML, searchStr); itemIdx != -1 {
				itemStart := strings.LastIndex(opfXML[:itemIdx], "<item")
				if itemStart != -1 {
					itemEnd := strings.Index(opfXML[itemStart:], "/>")
					if itemEnd == -1 {
						itemEnd = strings.Index(opfXML[itemStart:], "</item>")
					}
					if itemEnd != -1 {
						itemTag := opfXML[itemStart : itemStart+itemEnd]
						if strings.Contains(itemTag, "media-type=\"image/") {
							if hrefIdx := strings.Index(itemTag, "href=\""); hrefIdx != -1 {
								start := hrefIdx + 6
								end := strings.Index(itemTag[start:], "\"")
								if end != -1 {
									coverHref = itemTag[start : start+end]
								}
							}
						}
					}
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

	// Extract the cover image if found
	if coverHref != "" {
		var coverInZip string
		if opfDir != "" {
			coverInZip = filepath.Join(opfDir, coverHref)
		} else {
			coverInZip = coverHref
		}
		coverInZip = filepath.ToSlash(coverInZip)
		coverInZip = strings.ReplaceAll(coverInZip, "%20", " ")

		for _, f := range reader.File {
			if f.Name == coverInZip || filepath.ToSlash(f.Name) == coverInZip {
				rc, err := f.Open()
				if err != nil {
					continue
				}

				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					continue
				}

				if !IsImageFile(data) {
					log.Printf("Cover file %s is not a valid image", f.Name)
					continue
				}

				ext := ".jpg"
				if len(data) > 8 && data[0] == 0x89 && data[1] == 0x50 {
					ext = ".png"
				}

				if err := os.MkdirAll(coverDir, 0755); err != nil {
					continue
				}
				coverPath = filepath.Join(coverDir, "cover"+ext)
				outFile, err := os.Create(coverPath)
				if err != nil {
					continue
				}
				outFile.Write(data)
				outFile.Close()
				break
			}
		}
	}

	return
}

// ExtractPDFMetadata extracts title and author from a PDF file.
// originalName is used as the fallback title (should not include file extension).
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

	// Try to extract title from PDF metadata
	if idx := strings.Index(content, "/Title"); idx != -1 {
		if start := strings.Index(content[idx:], "("); start != -1 {
			start += idx + 1
			if end := strings.Index(content[start:], ")"); end != -1 && end < 200 {
				extractedTitle := content[start : start+end]
				extractedTitle = strings.TrimSpace(extractedTitle)
				extractedTitle = strings.ReplaceAll(extractedTitle, "\\(", "(")
				extractedTitle = strings.ReplaceAll(extractedTitle, "\\)", ")")
				if len(extractedTitle) > 2 && len(extractedTitle) < 200 {
					title = extractedTitle
				}
			}
		}
	}

	// Try to extract author from PDF metadata
	if idx := strings.Index(content, "/Author"); idx != -1 {
		if start := strings.Index(content[idx:], "("); start != -1 {
			start += idx + 1
			if end := strings.Index(content[start:], ")"); end != -1 && end < 200 {
				extractedAuthor := content[start : start+end]
				extractedAuthor = strings.TrimSpace(extractedAuthor)
				extractedAuthor = strings.ReplaceAll(extractedAuthor, "\\(", "(")
				extractedAuthor = strings.ReplaceAll(extractedAuthor, "\\)", ")")
				if len(extractedAuthor) > 0 && len(extractedAuthor) < 200 {
					author = extractedAuthor
				}
			}
		}
	}

	return
}

// ExtractPDFCover extracts the first page of a PDF as a JPEG cover image using pdftoppm.
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
		if err != nil || !IsImageFile(raw) {
			return
		}
		ext := ".jpg"
		if len(raw) > 8 && raw[0] == 0x89 && raw[1] == 0x50 {
			ext = ".png"
		}
		if err := os.MkdirAll(coverDir, 0755); err != nil {
			return
		}
		coverPath = filepath.Join(coverDir, "cover"+ext)
		if err := os.WriteFile(coverPath, raw, 0644); err != nil {
			coverPath = ""
		}
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
	if err != nil || !IsImageFile(data) {
		return ""
	}

	ext := ".jpg"
	if len(data) > 8 && data[0] == 0x89 && data[1] == 0x50 {
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

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
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
