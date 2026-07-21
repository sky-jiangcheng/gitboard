package main

import (
	"database/sql"
	"testing"

	"gitboard/internal/db"

	_ "modernc.org/sqlite"
)

// setupTestApp creates an in-memory App for integration testing.
func setupTestApp(t *testing.T) *App {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if _, err := database.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		root_path TEXT NOT NULL,
		level_override INTEGER DEFAULT 0,
		is_auto_grouped BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS project_todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		completed BOOLEAN DEFAULT 0,
		priority INTEGER DEFAULT 0,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS project_notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		title TEXT DEFAULT '',
		content TEXT NOT NULL,
		tags TEXT DEFAULT '',
		kind TEXT DEFAULT 'other',
		pinned INTEGER DEFAULT 0,
		source TEXT DEFAULT 'manual',
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);
	`
	if _, err := database.Exec(schema); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Create a test project
	res, err := database.Exec("INSERT INTO projects (name, root_path) VALUES (?, ?)", "test-project", "/tmp/test")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	pid, _ := res.LastInsertId()
	_ = pid

	return &App{db: database, gitUser: "testuser"}
}

func TestAppCreateAndListTodos(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	todo, err := app.CreateTodo(1, "Fix bug")
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}
	if todo.Title != "Fix bug" {
		t.Errorf("expected 'Fix bug', got '%s'", todo.Title)
	}

	todos := app.ListTodos(1)
	if len(todos) != 1 {
		t.Errorf("expected 1 todo, got %d", len(todos))
	}
}

func TestAppCreateTodoEmptyTitle(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	_, err := app.CreateTodo(1, "")
	if err == nil {
		t.Error("expected error for empty title")
	}
	_, err = app.CreateTodo(1, "   ")
	if err == nil {
		t.Error("expected error for whitespace-only title")
	}
}

func TestAppToggleTodo(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	todo, _ := app.CreateTodo(1, "Toggle me")
	if err := app.ToggleTodo(todo.ID); err != nil {
		t.Fatalf("ToggleTodo failed: %v", err)
	}

	todos := app.ListTodos(1)
	if !todos[0].Completed {
		t.Error("todo should be completed after toggle")
	}
}

func TestAppReorderTodos(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	t1, _ := app.CreateTodo(1, "A")
	t2, _ := app.CreateTodo(1, "B")
	t3, _ := app.CreateTodo(1, "C")

	if err := app.ReorderTodos([]int64{t3.ID, t2.ID, t1.ID}); err != nil {
		t.Fatalf("ReorderTodos failed: %v", err)
	}

	todos := app.ListTodos(1)
	if todos[0].Title != "C" {
		t.Errorf("first todo should be 'C', got '%s'", todos[0].Title)
	}
}

func TestAppDeleteTodo(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	todo, _ := app.CreateTodo(1, "Delete me")
	if err := app.DeleteTodo(todo.ID); err != nil {
		t.Fatalf("DeleteTodo failed: %v", err)
	}

	todos := app.ListTodos(1)
	if len(todos) != 0 {
		t.Errorf("expected 0 todos, got %d", len(todos))
	}
}

func TestAppCreateAndUpdateNote(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	note, err := app.CreateNote(1, "# Hello")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}
	if note.Content != "# Hello" {
		t.Errorf("expected '# Hello', got '%s'", note.Content)
	}

	if err := app.UpdateNote(note.ID, "# Updated"); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	notes := app.ListNotes(1)
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Content != "# Updated" {
		t.Errorf("expected '# Updated', got '%s'", notes[0].Content)
	}
}

func TestAppCreateNoteEmptyContent(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	_, err := app.CreateNote(1, "")
	if err == nil {
		t.Error("expected error for empty content")
	}
	err = app.UpdateNote(1, "   ")
	// UpdateNote on non-existent note returns error, but empty content check also rejects
	if err == nil {
		t.Error("expected error for empty content in update")
	}
}

func TestAppDeleteNote(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	note, _ := app.CreateNote(1, "Delete me")
	if err := app.DeleteNote(note.ID); err != nil {
		t.Fatalf("DeleteNote failed: %v", err)
	}

	notes := app.ListNotes(1)
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

func TestAppGetTodoCounts(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	app.CreateTodo(1, "T1") //nolint:errcheck
	app.CreateTodo(1, "T2") //nolint:errcheck
	t3, _ := app.CreateTodo(1, "T3")
	app.ToggleTodo(t3.ID) //nolint:errcheck

	counts := app.GetTodoCounts()
	if len(counts) != 1 {
		t.Fatalf("expected 1 count entry, got %d", len(counts))
	}
	if counts[0].Count != 2 || counts[0].Total != 3 {
		t.Errorf("expected count=2 total=3, got count=%d total=%d", counts[0].Count, counts[0].Total)
	}
}

func TestAppGetTodoCountsEmpty(t *testing.T) {
	app := setupTestApp(t)
	defer app.db.Close()

	counts := app.GetTodoCounts()
	if len(counts) != 0 {
		t.Errorf("expected empty counts, got %d", len(counts))
	}
}

// Ensure db.Todo and db.Note types are used for compilation check
var _ = db.Todo{}
var _ = db.Note{}
