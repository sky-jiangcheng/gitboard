package main

import (
	"fmt"
	"log"
	"strings"

	"gitboard/internal/db"
)

// NoteWithProject is a note joined with its parent project (global knowledge hub).
type NoteWithProject = db.NoteWithProject

// ListNotes returns all notes for a project.
func (a *App) ListNotes(projectID int64) []db.Note {
	notes, err := db.ListNotes(a.db, projectID)
	if err != nil {
		log.Printf("list notes error: %v", err)
		return nil
	}
	if notes == nil {
		notes = []db.Note{}
	}
	return notes
}

// CreateNote creates a new note for a project.
func (a *App) CreateNote(projectID int64, content string) (*db.Note, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	return db.CreateNote(a.db, projectID, content)
}

// UpdateNote updates the content of a note.
func (a *App) UpdateNote(noteID int64, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is required")
	}
	return db.UpdateNote(a.db, noteID, content)
}

// DeleteNote removes a note.
func (a *App) DeleteNote(noteID int64) error {
	return db.DeleteNote(a.db, noteID)
}

// ListAllNotes returns every note across all projects, joined with project info,
// ordered pinned first then most recently updated.
func (a *App) ListAllNotes() []NoteWithProject {
	notes, err := db.ListAllNotes(a.db)
	if err != nil {
		log.Printf("list all notes error: %v", err)
		return nil
	}
	if notes == nil {
		notes = []db.NoteWithProject{}
	}
	return notes
}

// ListAllTags returns the distinct set of tags used across all notes.
func (a *App) ListAllTags() []string {
	tags, err := db.ListAllTags(a.db)
	if err != nil {
		log.Printf("list all tags error: %v", err)
		return nil
	}
	if tags == nil {
		tags = []string{}
	}
	return tags
}

// CreateNoteWithMeta creates a note with explicit title, tags, kind, and source.
func (a *App) CreateNoteWithMeta(projectID int64, title, content, tags, kind, source string) (*db.Note, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	return db.CreateNoteEx(a.db, projectID, title, content, tags, kind, source)
}

// UpdateNoteMeta updates a note's editable metadata (title, tags, kind, pinned).
func (a *App) UpdateNoteMeta(noteID int64, title, tags, kind string, pinned bool) error {
	return db.UpdateNoteMeta(a.db, noteID, title, tags, kind, pinned)
}

// PinNote sets or clears the pinned flag on a note.
func (a *App) PinNote(noteID int64, pinned bool) error {
	return db.PinNote(a.db, noteID, pinned)
}
