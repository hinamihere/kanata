package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"kana/core"

	_ "modernc.org/sqlite"
)

const (
	KanaDirName = ".kana"
	DBFileName  = "metadata.db"
)

// Snapshot models a point-in-time semantic state of the repository.
type Snapshot struct {
	Hash       string    `json:"hash"`
	ParentHash string    `json:"parent_hash"`
	WorkStream string    `json:"work_stream"`
	Author     string    `json:"author"`
	Intent     string    `json:"intent"`
	Timestamp  time.Time `json:"timestamp"`
	TreeHash   string    `json:"tree_hash"`
}

// ParkedState holds temporarily parked workspace AST nodes.
type ParkedState struct {
	ID         string    `json:"id"`
	WorkStream string    `json:"work_stream"`
	Note       string    `json:"note"`
	Timestamp  time.Time `json:"timestamp"`
	Data       string    `json:"data"`
}

// Storage manages the embedded SQLite database for Kanata.
type Storage struct {
	db       *sql.DB
	RepoPath string
	KanaDir  string
}

// ComputeTreeHash generates a deterministic SHA-256 hash across all workspace file ASTs.
func ComputeTreeHash(files map[string]*core.FileAST) string {
	var keys []string
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		f := files[k]
		h.Write([]byte(k + ":" + f.RawHash + ";"))
		for _, n := range f.SortedNodes() {
			h.Write([]byte(n.ID + "=" + n.Hash + ";"))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ComputeSnapshotHash computes the unique cryptographic ID for a snapshot.
func ComputeSnapshotHash(parentHash, workStream, author, intent string, t time.Time, treeHash string) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%d|%s", parentHash, workStream, author, intent, t.UnixNano(), treeHash)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// InitRepo initializes the .kana metadata directory and database schema.
func InitRepo(repoPath string) (*Storage, error) {
	kanaDir := filepath.Join(repoPath, KanaDirName)
	if err := os.MkdirAll(kanaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", kanaDir, err)
	}

	dbPath := filepath.Join(kanaDir, DBFileName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{
		db:       db,
		RepoPath: repoPath,
		KanaDir:  kanaDir,
	}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Set default stream "main"
	_ = s.SetConfig("current_stream", "main")
	_ = s.CreateStream("main", "")

	return s, nil
}

// OpenRepo opens an existing .kana metadata repository.
func OpenRepo(startPath string) (*Storage, error) {
	curr := startPath
	for {
		candidate := filepath.Join(curr, KanaDirName)
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			dbPath := filepath.Join(candidate, DBFileName)
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open database at %s: %w", dbPath, err)
			}
			s := &Storage{
				db:       db,
				RepoPath: curr,
				KanaDir:  candidate,
			}
			return s, nil
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return nil, fmt.Errorf("no kanata repository found (run 'kana init' to create one)")
}

// Close releases the database handle.
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS streams (
		name TEXT PRIMARY KEY,
		head_snapshot_hash TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS snapshots (
		hash TEXT PRIMARY KEY,
		parent_hash TEXT,
		work_stream TEXT NOT NULL,
		author TEXT NOT NULL,
		intent TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		tree_hash TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS snapshot_nodes (
		snapshot_hash TEXT NOT NULL,
		file_path TEXT NOT NULL,
		node_id TEXT NOT NULL,
		node_name TEXT NOT NULL,
		signature TEXT NOT NULL,
		node_type TEXT NOT NULL,
		language TEXT NOT NULL,
		start_line INTEGER,
		end_line INTEGER,
		content TEXT,
		node_hash TEXT NOT NULL,
		doc_comment TEXT,
		raw_file_hash TEXT NOT NULL,
		PRIMARY KEY (snapshot_hash, file_path, node_id)
	);

	CREATE TABLE IF NOT EXISTS parked_states (
		id TEXT PRIMARY KEY,
		work_stream TEXT NOT NULL,
		note TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		data TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// GetConfig reads a configuration key.
func (s *Storage) GetConfig(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetConfig writes or updates a configuration key.
func (s *Storage) SetConfig(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

// GetCurrentStream returns the active work stream name.
func (s *Storage) GetCurrentStream() (string, error) {
	val, err := s.GetConfig("current_stream")
	if err != nil || val == "" {
		return "main", nil
	}
	return val, nil
}

// SetCurrentStream updates the active work stream.
func (s *Storage) SetCurrentStream(stream string) error {
	return s.SetConfig("current_stream", stream)
}

// CreateStream creates a new work stream.
func (s *Storage) CreateStream(name string, baseSnapshotHash string) error {
	_, err := s.db.Exec(`
		INSERT INTO streams (name, head_snapshot_hash, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO NOTHING
	`, name, baseSnapshotHash, time.Now().UTC())
	return err
}

// ListStreams returns all registered work stream names.
func (s *Storage) ListStreams() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM streams ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		streams = append(streams, name)
	}
	return streams, nil
}

// GetStreamHead retrieves the latest snapshot hash for a stream.
func (s *Storage) GetStreamHead(stream string) (string, error) {
	var head sql.NullString
	err := s.db.QueryRow("SELECT head_snapshot_hash FROM streams WHERE name = ?", stream).Scan(&head)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if head.Valid {
		return head.String, nil
	}
	return "", nil
}

// SetStreamHead sets the latest snapshot hash for a stream.
func (s *Storage) SetStreamHead(stream, snapshotHash string) error {
	_, err := s.db.Exec(`
		INSERT INTO streams (name, head_snapshot_hash, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET head_snapshot_hash = excluded.head_snapshot_hash
	`, stream, snapshotHash, time.Now().UTC())
	return err
}

// SaveSnapshot persists a snapshot and all its structural AST nodes.
func (s *Storage) SaveSnapshot(snap *Snapshot, files map[string]*core.FileAST) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO snapshots (hash, parent_hash, work_stream, author, intent, timestamp, tree_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, snap.Hash, snap.ParentHash, snap.WorkStream, snap.Author, snap.Intent, snap.Timestamp, snap.TreeHash)
	if err != nil {
		return fmt.Errorf("failed to insert snapshot: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO snapshot_nodes (
			snapshot_hash, file_path, node_id, node_name, signature, node_type, language,
			start_line, end_line, content, node_hash, doc_comment, raw_file_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare node insert: %w", err)
	}
	defer stmt.Close()

	for filePath, fAST := range files {
		for _, node := range fAST.Nodes {
			_, err = stmt.Exec(
				snap.Hash,
				filePath,
				node.ID,
				node.Name,
				node.Signature,
				string(node.Type),
				node.Language,
				node.StartLine,
				node.EndLine,
				node.Content,
				node.Hash,
				node.DocComment,
				fAST.RawHash,
			)
			if err != nil {
				return fmt.Errorf("failed to save node %s: %w", node.ID, err)
			}
		}
	}

	// Update stream head
	_, err = tx.Exec(`
		UPDATE streams SET head_snapshot_hash = ? WHERE name = ?
	`, snap.Hash, snap.WorkStream)
	if err != nil {
		return fmt.Errorf("failed to update stream head: %w", err)
	}

	return tx.Commit()
}

// GetSnapshot retrieves metadata for a specific snapshot hash.
func (s *Storage) GetSnapshot(hash string) (*Snapshot, error) {
	if hash == "" {
		return nil, nil
	}
	var snap Snapshot
	var parent sql.NullString
	err := s.db.QueryRow(`
		SELECT hash, parent_hash, work_stream, author, intent, timestamp, tree_hash
		FROM snapshots WHERE hash = ? OR hash LIKE ?
	`, hash, hash+"%").Scan(
		&snap.Hash,
		&parent,
		&snap.WorkStream,
		&snap.Author,
		&snap.Intent,
		&snap.Timestamp,
		&snap.TreeHash,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("snapshot '%s' not found", hash)
	}
	if err != nil {
		return nil, err
	}
	if parent.Valid {
		snap.ParentHash = parent.String
	}
	return &snap, nil
}

// GetSnapshotAST reconstructs the full workspace FileAST map from a snapshot.
func (s *Storage) GetSnapshotAST(snapshotHash string) (map[string]*core.FileAST, error) {
	snap, err := s.GetSnapshot(snapshotHash)
	if err != nil || snap == nil {
		return make(map[string]*core.FileAST), nil
	}

	rows, err := s.db.Query(`
		SELECT file_path, node_id, node_name, signature, node_type, language,
		       start_line, end_line, content, node_hash, doc_comment, raw_file_hash
		FROM snapshot_nodes WHERE snapshot_hash = ?
	`, snap.Hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*core.FileAST)
	for rows.Next() {
		var (
			filePath, nodeID, nodeName, sig, nType, lang, content, nodeHash, doc, rawHash string
			startLine, endLine                                                             int
		)
		err := rows.Scan(
			&filePath, &nodeID, &nodeName, &sig, &nType, &lang,
			&startLine, &endLine, &content, &nodeHash, &doc, &rawHash,
		)
		if err != nil {
			return nil, err
		}

		fAST, ok := result[filePath]
		if !ok {
			fAST = &core.FileAST{
				FilePath: filePath,
				Language: lang,
				Nodes:    make(map[string]core.ASTNode),
				RawHash:  rawHash,
			}
			result[filePath] = fAST
		}

		fAST.Nodes[nodeID] = core.ASTNode{
			ID:         nodeID,
			Name:       nodeName,
			Signature:  sig,
			Type:       core.NodeType(nType),
			Language:   lang,
			StartLine:  startLine,
			EndLine:    endLine,
			Content:    content,
			Hash:       nodeHash,
			DocComment: doc,
		}
	}

	return result, nil
}

// ListSnapshots returns snapshots in descending chronological order for a stream.
func (s *Storage) ListSnapshots(stream string, limit int) ([]*Snapshot, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT hash, parent_hash, work_stream, author, intent, timestamp, tree_hash
		FROM snapshots
	`
	var args []interface{}
	if stream != "" {
		query += " WHERE work_stream = ?"
		args = append(args, stream)
	}
	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Snapshot
	for rows.Next() {
		var snap Snapshot
		var parent sql.NullString
		err := rows.Scan(
			&snap.Hash,
			&parent,
			&snap.WorkStream,
			&snap.Author,
			&snap.Intent,
			&snap.Timestamp,
			&snap.TreeHash,
		)
		if err != nil {
			return nil, err
		}
		if parent.Valid {
			snap.ParentHash = parent.String
		}
		list = append(list, &snap)
	}

	return list, nil
}

// ParkWorkspace serializes current workspace AST and parks it out of band.
func (s *Storage) ParkWorkspace(stream, note string, fileMap map[string]*core.FileAST) (string, error) {
	dataBytes, err := json.Marshal(fileMap)
	if err != nil {
		return "", err
	}

	t := time.Now().UTC()
	idRaw := fmt.Sprintf("park-%s-%d", stream, t.UnixNano())
	idHash := sha256.Sum256([]byte(idRaw))
	id := hex.EncodeToString(idHash[:8])

	_, err = s.db.Exec(`
		INSERT INTO parked_states (id, work_stream, note, timestamp, data)
		VALUES (?, ?, ?, ?, ?)
	`, id, stream, note, t, string(dataBytes))
	if err != nil {
		return "", err
	}

	return id, nil
}

// ListParked retrieves all parked states.
func (s *Storage) ListParked(stream string) ([]*ParkedState, error) {
	query := "SELECT id, work_stream, note, timestamp, data FROM parked_states"
	var args []interface{}
	if stream != "" {
		query += " WHERE work_stream = ?"
		args = append(args, stream)
	}
	query += " ORDER BY timestamp DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ParkedState
	for rows.Next() {
		var ps ParkedState
		err := rows.Scan(&ps.ID, &ps.WorkStream, &ps.Note, &ps.Timestamp, &ps.Data)
		if err != nil {
			return nil, err
		}
		result = append(result, &ps)
	}
	return result, nil
}

// PopParked retrieves and removes the latest parked state.
func (s *Storage) PopParked(id string) (*ParkedState, map[string]*core.FileAST, error) {
	var ps ParkedState
	var err error
	if id != "" {
		err = s.db.QueryRow("SELECT id, work_stream, note, timestamp, data FROM parked_states WHERE id = ?", id).
			Scan(&ps.ID, &ps.WorkStream, &ps.Note, &ps.Timestamp, &ps.Data)
	} else {
		err = s.db.QueryRow("SELECT id, work_stream, note, timestamp, data FROM parked_states ORDER BY timestamp DESC LIMIT 1").
			Scan(&ps.ID, &ps.WorkStream, &ps.Note, &ps.Timestamp, &ps.Data)
	}

	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("no parked states found")
	}
	if err != nil {
		return nil, nil, err
	}

	var files map[string]*core.FileAST
	if err := json.Unmarshal([]byte(ps.Data), &files); err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize parked state: %w", err)
	}

	_, _ = s.db.Exec("DELETE FROM parked_states WHERE id = ?", ps.ID)
	return &ps, files, nil
}
