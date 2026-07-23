package db

import (
	"database/sql"
	"testing"
)

// setupTestDB creates an in-memory SQLite database with all tables for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}
	// Create all tables
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		root_path TEXT NOT NULL,
		level_override INTEGER DEFAULT 0,
		is_auto_grouped BOOLEAN DEFAULT 1,
		is_starred INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS repositories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		project_id INTEGER,
		last_scanned_at DATETIME,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	CREATE TABLE IF NOT EXISTS daily_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repository_id INTEGER NOT NULL,
		stat_date DATE NOT NULL,
		author TEXT NOT NULL,
		files_changed INTEGER DEFAULT 0,
		lines_added INTEGER DEFAULT 0,
		lines_deleted INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (repository_id) REFERENCES repositories(id),
		UNIQUE(repository_id, stat_date, author)
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
	CREATE TABLE IF NOT EXISTS repo_meta (
		repository_id INTEGER PRIMARY KEY,
		tech_stack TEXT DEFAULT '[]',
		readme_excerpt TEXT DEFAULT '',
		languages TEXT DEFAULT '{}',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	return db
}

// createTestProject inserts a project and returns its ID.
func createTestProject(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	res, err := db.Exec(
		"INSERT INTO projects (name, root_path) VALUES (?, ?)",
		name, "/tmp/"+name,
	)
	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get project id: %v", err)
	}
	return id
}

// -- Todo tests --

func TestCreateAndListTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "test-project")

	// Create two todos
	t1, err := CreateTodo(db, pid, "Fix login bug")
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}
	if t1.Title != "Fix login bug" || t1.Completed {
		t.Errorf("unexpected todo values: %+v", t1)
	}

	t2, err := CreateTodo(db, pid, "Add unit tests")
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}
	_ = t2

	// List should return both in order
	todos, err := ListTodos(db, pid)
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("expected 2 todos, got %d", len(todos))
	}
	if todos[0].Title != "Fix login bug" {
		t.Errorf("first todo should be 'Fix login bug', got '%s'", todos[0].Title)
	}
	if todos[1].Title != "Add unit tests" {
		t.Errorf("second todo should be 'Add unit tests', got '%s'", todos[1].Title)
	}
	// sort_order should be sequential
	if todos[0].SortOrder != 0 || todos[1].SortOrder != 1 {
		t.Errorf("unexpected sort_order: %d, %d", todos[0].SortOrder, todos[1].SortOrder)
	}
}

func TestToggleTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "toggle-test")

	todo, err := CreateTodo(db, pid, "Test toggle")
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	if todo.Completed {
		t.Error("new todo should not be completed")
	}

	// Toggle to completed
	if err := ToggleTodo(db, todo.ID); err != nil {
		t.Fatalf("ToggleTodo failed: %v", err)
	}

	todos, err := ListTodos(db, pid)
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}
	if !todos[0].Completed {
		t.Error("todo should be completed after toggle")
	}

	// Toggle back to incomplete
	if err := ToggleTodo(db, todo.ID); err != nil {
		t.Fatalf("second ToggleTodo failed: %v", err)
	}
	todos, err = ListTodos(db, pid)
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}
	if todos[0].Completed {
		t.Error("todo should be incomplete after second toggle")
	}
}

func TestToggleNonExistentTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := ToggleTodo(db, 99999)
	if err == nil {
		t.Error("expected error when toggling non-existent todo")
	}
}

func TestDeleteTodo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "delete-test")

	todo, err := CreateTodo(db, pid, "Delete me")
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}

	if err := DeleteTodo(db, todo.ID); err != nil {
		t.Fatalf("DeleteTodo failed: %v", err)
	}

	todos, err := ListTodos(db, pid)
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos after delete, got %d", len(todos))
	}

	// Deleting a non-existent todo should not error
	if err := DeleteTodo(db, 99999); err != nil {
		t.Errorf("DeleteTodo on non-existent id should not error: %v", err)
	}
}

func TestReorderTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "reorder-test")

	// Create 3 todos
	t1, _ := CreateTodo(db, pid, "A")
	t2, _ := CreateTodo(db, pid, "B")
	t3, _ := CreateTodo(db, pid, "C")

	// Reverse the order
	if err := ReorderTodos(db, []int64{t3.ID, t2.ID, t1.ID}); err != nil {
		t.Fatalf("ReorderTodos failed: %v", err)
	}

	todos, err := ListTodos(db, pid)
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}
	if len(todos) != 3 {
		t.Fatalf("expected 3 todos, got %d", len(todos))
	}
	// After reorder, C should be first, A should be last
	if todos[0].Title != "C" {
		t.Errorf("first todo should be 'C', got '%s'", todos[0].Title)
	}
	if todos[2].Title != "A" {
		t.Errorf("last todo should be 'A', got '%s'", todos[2].Title)
	}
}

func TestListTodosEmptyProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "empty-project")

	todos, err := ListTodos(db, pid)
	if err != nil {
		t.Fatalf("ListTodos failed: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected empty list, got %d items", len(todos))
	}
}

// -- Note tests --

func TestCreateAndListNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "note-test")

	n1, err := CreateNote(db, pid, "# Meeting notes\n- Discussed architecture")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}
	if n1.Content != "# Meeting notes\n- Discussed architecture" {
		t.Errorf("unexpected note content: %s", n1.Content)
	}
	if n1.CreatedAt != n1.UpdatedAt {
		t.Error("new note should have same created_at and updated_at")
	}

	notes, err := ListNotes(db, pid)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
}

func TestUpdateNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "update-note-test")

	note, err := CreateNote(db, pid, "Original content")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	originalUpdatedAt := note.UpdatedAt

	if err := UpdateNote(db, note.ID, "Updated content"); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}

	notes, err := ListNotes(db, pid)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Content != "Updated content" {
		t.Errorf("expected 'Updated content', got '%s'", notes[0].Content)
	}
	if notes[0].UpdatedAt == originalUpdatedAt {
		t.Error("updated_at should change after update")
	}
}

func TestUpdateNonExistentNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	err := UpdateNote(db, 99999, "content")
	if err == nil {
		t.Error("expected error when updating non-existent note")
	}
}

func TestDeleteNote(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "delete-note-test")

	note, err := CreateNote(db, pid, "Delete me")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	if err := DeleteNote(db, note.ID); err != nil {
		t.Fatalf("DeleteNote failed: %v", err)
	}

	notes, err := ListNotes(db, pid)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes after delete, got %d", len(notes))
	}
}

func TestNotesEmptyProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "empty-note-project")

	notes, err := ListNotes(db, pid)
	if err != nil {
		t.Fatalf("ListNotes failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected empty notes list, got %d", len(notes))
	}
}

// -- CASCADE delete test --

func TestCascadeDeleteProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	pid := createTestProject(t, db, "cascade-test")

	// Create todo and note
	_, err := CreateTodo(db, pid, "Todo under cascade")
	if err != nil {
		t.Fatalf("CreateTodo failed: %v", err)
	}
	_, err = CreateNote(db, pid, "Note under cascade")
	if err != nil {
		t.Fatalf("CreateNote failed: %v", err)
	}

	// Delete the project
	_, err = db.Exec("DELETE FROM projects WHERE id = ?", pid)
	if err != nil {
		t.Fatalf("delete project failed: %v", err)
	}

	// Todos and notes should be cascade-deleted
	todos, _ := ListTodos(db, pid)
	if len(todos) != 0 {
		t.Errorf("expected 0 todos after cascade delete, got %d", len(todos))
	}

	notes, _ := ListNotes(db, pid)
	if len(notes) != 0 {
		t.Errorf("expected 0 notes after cascade delete, got %d", len(notes))
	}
}

// -- GetTodoCounts test --

func TestGetTodoCounts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	p1 := createTestProject(t, db, "counts-p1")
	p2 := createTestProject(t, db, "counts-p2")

	// Project 1: 3 todos, 1 completed
	CreateTodo(db, p1, "T1") //nolint:errcheck
	CreateTodo(db, p1, "T2") //nolint:errcheck
	t3, _ := CreateTodo(db, p1, "T3")
	ToggleTodo(db, t3.ID) //nolint:errcheck

	// Project 2: 1 todo, 0 completed
	CreateTodo(db, p2, "T4") //nolint:errcheck

	counts, err := GetTodoCounts(db)
	if err != nil {
		t.Fatalf("GetTodoCounts failed: %v", err)
	}

	if len(counts) != 2 {
		t.Fatalf("expected 2 project counts, got %d", len(counts))
	}

	for _, c := range counts {
		switch c.ProjectID {
		case p1:
			if c.Count != 2 || c.Total != 3 {
				t.Errorf("project 1: expected count=2 total=3, got count=%d total=%d", c.Count, c.Total)
			}
		case p2:
			if c.Count != 1 || c.Total != 1 {
				t.Errorf("project 2: expected count=1 total=1, got count=%d total=%d", c.Count, c.Total)
			}
		default:
			t.Errorf("unexpected project ID: %d", c.ProjectID)
		}
	}
}

func TestGetTodoCountsEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	counts, err := GetTodoCounts(db)
	if err != nil {
		t.Fatalf("GetTodoCounts failed: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty counts, got %d", len(counts))
	}
}

// -- SyncProjectTx tests --

func TestSyncProjectTx_CreateNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck

	id, err := SyncProjectTx(tx, "my-project", "/repos/my-project", 0, true)
	if err != nil {
		t.Fatalf("SyncProjectTx failed: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero project ID")
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Verify the project was created
	p, err := GetProjectByID(db, id)
	if err != nil {
		t.Fatalf("GetProjectByID failed: %v", err)
	}
	if p.Name != "my-project" || p.RootPath != "/repos/my-project" {
		t.Errorf("unexpected project: %+v", p)
	}
}

func TestSyncProjectTx_PreservesExisting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a project with is_starred = 1 and is_auto_grouped = 0 (manually adjusted)
	res, err := db.Exec(
		"INSERT INTO projects (name, root_path, is_auto_grouped, is_starred) VALUES (?, ?, ?, ?)",
		"old-name", "/repos/proj", 0, 1,
	)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	origID, _ := res.LastInsertId()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Sync with new name and is_auto_grouped=true
	id, err := SyncProjectTx(tx, "new-name", "/repos/proj", 0, true)
	if err != nil {
		t.Fatalf("SyncProjectTx failed: %v", err)
	}
	if id != origID {
		t.Errorf("expected same project ID %d, got %d", origID, id)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// is_starred should be preserved, is_auto_grouped should stay 0 (manually adjusted)
	p, err := GetProjectByID(db, id)
	if err != nil {
		t.Fatalf("GetProjectByID failed: %v", err)
	}
	if !p.IsStarred {
		t.Error("is_starred should be preserved as true")
	}
	if p.IsAutoGrouped {
		t.Error("is_auto_grouped should remain false for manually adjusted projects")
	}
}

func TestSyncProjectTx_UpdatesAutoGrouped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create an auto-grouped project
	res, _ := db.Exec(
		"INSERT INTO projects (name, root_path, is_auto_grouped, is_starred) VALUES (?, ?, ?, ?)",
		"old-name", "/repos/proj", 1, 0,
	)
	origID, _ := res.LastInsertId()

	tx, _ := db.Begin()
	defer tx.Rollback() //nolint:errcheck

	id, err := SyncProjectTx(tx, "new-name", "/repos/proj", 0, true)
	if err != nil {
		t.Fatalf("SyncProjectTx failed: %v", err)
	}
	tx.Commit() //nolint:errcheck

	p, _ := GetProjectByID(db, id)
	if p.Name != "new-name" {
		t.Errorf("expected name 'new-name', got '%s'", p.Name)
	}
	if p.IsAutoGrouped != true {
		t.Error("is_auto_grouped should be updated to true for auto-grouped projects")
	}
	if p.IsStarred {
		t.Error("is_starred should remain false")
	}
	if id != origID {
		t.Errorf("expected same ID, got different")
	}
}

// -- CleanupStaleDataTx tests --

func TestCleanupStaleDataTx_RemovesStaleRepos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create two projects with repos
	p1 := createTestProject(t, db, "proj1")
	p2 := createTestProject(t, db, "proj2")

	db.Exec("INSERT INTO repositories (path, project_id) VALUES (?, ?)", "/repos/p1", p1) //nolint:errcheck
	db.Exec("INSERT INTO repositories (path, project_id) VALUES (?, ?)", "/repos/p2", p2) //nolint:errcheck

	// Add stats for both repos
	var r1ID, r2ID int64
	db.QueryRow("SELECT id FROM repositories WHERE path = '/repos/p1'").Scan(&r1ID)
	db.QueryRow("SELECT id FROM repositories WHERE path = '/repos/p2'").Scan(&r2ID)
	db.Exec("INSERT INTO daily_stats (repository_id, stat_date, author, lines_added) VALUES (?, ?, ?, ?)",
		r1ID, "2024-01-01", "all", 100) //nolint:errcheck
	db.Exec("INSERT INTO daily_stats (repository_id, stat_date, author, lines_added) VALUES (?, ?, ?, ?)",
		r2ID, "2024-01-01", "all", 200) //nolint:errcheck

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Only p1's repo is in the scanned set
	err = CleanupStaleDataTx(tx, []string{"/repos/p1"})
	if err != nil {
		t.Fatalf("CleanupStaleDataTx failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// p2's repo should be deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM repositories WHERE path = '/repos/p2'").Scan(&count)
	if count != 0 {
		t.Error("stale repo should be deleted")
	}

	// p1's repo should still exist
	db.QueryRow("SELECT COUNT(*) FROM repositories WHERE path = '/repos/p1'").Scan(&count)
	if count != 1 {
		t.Error("scanned repo should still exist")
	}

	// p2's stats should be deleted
	db.QueryRow("SELECT COUNT(*) FROM daily_stats WHERE repository_id = ?", r2ID).Scan(&count)
	if count != 0 {
		t.Error("stale repo stats should be deleted")
	}
}

func TestCleanupStaleDataTx_PreservesProjectsWithNotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a project with a note but no repos (simulating a project whose repos disappeared)
	p1 := createTestProject(t, db, "proj-with-notes")
	CreateNote(db, p1, "important note") //nolint:errcheck

	// Create a project with no notes/repos (orphaned)
	p2 := createTestProject(t, db, "orphaned")

	tx, _ := db.Begin()
	defer tx.Rollback() //nolint:errcheck

	// Empty scanned paths - all repos are stale
	err := CleanupStaleDataTx(tx, []string{"/nonexistent"})
	if err != nil {
		t.Fatalf("CleanupStaleDataTx failed: %v", err)
	}
	tx.Commit() //nolint:errcheck

	// Project with notes should be preserved
	var count int
	db.QueryRow("SELECT COUNT(*) FROM projects WHERE id = ?", p1).Scan(&count)
	if count != 1 {
		t.Error("project with notes should be preserved")
	}

	// Orphaned project should be deleted
	db.QueryRow("SELECT COUNT(*) FROM projects WHERE id = ?", p2).Scan(&count)
	if count != 0 {
		t.Error("orphaned project should be deleted")
	}
}

func TestCleanupStaleDataTx_EmptyPaths(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tx, _ := db.Begin()
	defer tx.Rollback() //nolint:errcheck

	// Should be a no-op with empty paths
	err := CleanupStaleDataTx(tx, []string{})
	if err != nil {
		t.Fatalf("CleanupStaleDataTx with empty paths should not error: %v", err)
	}
}
