package handlers

import (
	"quiret/db"
	"quiret/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func GetAnnotations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookID := vars["id"]

	rows, err := db.DB.Query(
		"SELECT id, book_id, cfi, text, note, color, created_at FROM annotations WHERE book_id = ? ORDER BY created_at DESC",
		bookID,
	)
	if err != nil {
		http.Error(w, "Failed to fetch annotations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	annotations := make([]models.Annotation, 0)
	for rows.Next() {
		var a models.Annotation
		var note *string
		err := rows.Scan(&a.ID, &a.BookID, &a.CFI, &a.Text, &note, &a.Color, &a.CreatedAt)
		if err != nil {
			continue
		}
		if note != nil {
			a.Note = *note
		}
		annotations = append(annotations, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(annotations)
}

func CreateAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookID := vars["id"]

	var input struct {
		CFI   string `json:"cfi"`
		Text  string `json:"text"`
		Note  string `json:"note"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.CFI == "" || input.Text == "" {
		http.Error(w, "CFI and text are required", http.StatusBadRequest)
		return
	}

	if input.Color == "" {
		input.Color = "yellow"
	}

	annotation := models.Annotation{
		ID:        uuid.New().String(),
		BookID:    bookID,
		CFI:       input.CFI,
		Text:      input.Text,
		Note:      input.Note,
		Color:     input.Color,
		CreatedAt: time.Now(),
	}

	_, err := db.DB.Exec(
		"INSERT INTO annotations (id, book_id, cfi, text, note, color, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		annotation.ID, annotation.BookID, annotation.CFI, annotation.Text, annotation.Note, annotation.Color, annotation.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Failed to create annotation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(annotation)
}

func UpdateAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	annotationID := vars["annotationId"]

	var input struct {
		Note  string `json:"note"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err := db.DB.Exec(
		"UPDATE annotations SET note = ?, color = ? WHERE id = ?",
		input.Note, input.Color, annotationID,
	)
	if err != nil {
		http.Error(w, "Failed to update annotation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func DeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	annotationID := vars["annotationId"]

	_, err := db.DB.Exec("DELETE FROM annotations WHERE id = ?", annotationID)
	if err != nil {
		http.Error(w, "Failed to delete annotation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
