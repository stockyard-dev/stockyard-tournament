package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Tournaments struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Game string `json:"game"`
	Format string `json:"format"`
	Date string `json:"date"`
	Location string `json:"location"`
	MaxParticipants int64 `json:"max_participants"`
	Status string `json:"status"`
	Notes string `json:"notes"`
	CreatedAt string `json:"created_at"`
}

type Participants struct {
	ID string `json:"id"`
	TournamentId string `json:"tournament_id"`
	Name string `json:"name"`
	Email string `json:"email"`
	Seed int64 `json:"seed"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type Matches struct {
	ID string `json:"id"`
	TournamentId string `json:"tournament_id"`
	Round int64 `json:"round"`
	MatchNumber int64 `json:"match_number"`
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	Score1 string `json:"score1"`
	Score2 string `json:"score2"`
	Winner string `json:"winner"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil { return nil, err }
	db, err := sql.Open("sqlite", filepath.Join(d, "tournament.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil { return nil, err }
	db.SetMaxOpenConns(1)
	db.Exec(`CREATE TABLE IF NOT EXISTS tournaments(id TEXT PRIMARY KEY, name TEXT NOT NULL, game TEXT DEFAULT '', format TEXT DEFAULT '', date TEXT DEFAULT '', location TEXT DEFAULT '', max_participants INTEGER DEFAULT 0, status TEXT DEFAULT '', notes TEXT DEFAULT '', created_at TEXT DEFAULT(datetime('now')))`)
	db.Exec(`CREATE TABLE IF NOT EXISTS participants(id TEXT PRIMARY KEY, tournament_id TEXT NOT NULL, name TEXT NOT NULL, email TEXT DEFAULT '', seed INTEGER DEFAULT 0, status TEXT DEFAULT '', created_at TEXT DEFAULT(datetime('now')))`)
	db.Exec(`CREATE TABLE IF NOT EXISTS matches(id TEXT PRIMARY KEY, tournament_id TEXT NOT NULL, round INTEGER DEFAULT 0, match_number INTEGER DEFAULT 0, player1 TEXT DEFAULT '', player2 TEXT DEFAULT '', score1 TEXT DEFAULT '', score2 TEXT DEFAULT '', winner TEXT DEFAULT '', status TEXT DEFAULT '', created_at TEXT DEFAULT(datetime('now')))`)
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL, record_id TEXT NOT NULL, data TEXT NOT NULL DEFAULT '{}', PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) CreateTournaments(e *Tournaments) error {
	e.ID = genID(); e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO tournaments(id, name, game, format, date, location, max_participants, status, notes, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, e.Name, e.Game, e.Format, e.Date, e.Location, e.MaxParticipants, e.Status, e.Notes, e.CreatedAt)
	return err
}

func (d *DB) GetTournaments(id string) *Tournaments {
	var e Tournaments
	if d.db.QueryRow(`SELECT id, name, game, format, date, location, max_participants, status, notes, created_at FROM tournaments WHERE id=?`, id).Scan(&e.ID, &e.Name, &e.Game, &e.Format, &e.Date, &e.Location, &e.MaxParticipants, &e.Status, &e.Notes, &e.CreatedAt) != nil { return nil }
	return &e
}

func (d *DB) ListTournaments() []Tournaments {
	rows, _ := d.db.Query(`SELECT id, name, game, format, date, location, max_participants, status, notes, created_at FROM tournaments ORDER BY created_at DESC`)
	if rows == nil { return nil }; defer rows.Close()
	var o []Tournaments
	for rows.Next() { var e Tournaments; rows.Scan(&e.ID, &e.Name, &e.Game, &e.Format, &e.Date, &e.Location, &e.MaxParticipants, &e.Status, &e.Notes, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) UpdateTournaments(e *Tournaments) error {
	_, err := d.db.Exec(`UPDATE tournaments SET name=?, game=?, format=?, date=?, location=?, max_participants=?, status=?, notes=? WHERE id=?`, e.Name, e.Game, e.Format, e.Date, e.Location, e.MaxParticipants, e.Status, e.Notes, e.ID)
	return err
}

func (d *DB) DeleteTournaments(id string) error {
	_, err := d.db.Exec(`DELETE FROM tournaments WHERE id=?`, id)
	return err
}

func (d *DB) CountTournaments() int {
	var n int; d.db.QueryRow(`SELECT COUNT(*) FROM tournaments`).Scan(&n); return n
}

func (d *DB) SearchTournaments(q string, filters map[string]string) []Tournaments {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (name LIKE ? OR game LIKE ? OR location LIKE ? OR notes LIKE ?)"
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["format"]; ok && v != "" { where += " AND format=?"; args = append(args, v) }
	if v, ok := filters["status"]; ok && v != "" { where += " AND status=?"; args = append(args, v) }
	rows, _ := d.db.Query(`SELECT id, name, game, format, date, location, max_participants, status, notes, created_at FROM tournaments WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil { return nil }; defer rows.Close()
	var o []Tournaments
	for rows.Next() { var e Tournaments; rows.Scan(&e.ID, &e.Name, &e.Game, &e.Format, &e.Date, &e.Location, &e.MaxParticipants, &e.Status, &e.Notes, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) CreateParticipants(e *Participants) error {
	e.ID = genID(); e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO participants(id, tournament_id, name, email, seed, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, e.ID, e.TournamentId, e.Name, e.Email, e.Seed, e.Status, e.CreatedAt)
	return err
}

func (d *DB) GetParticipants(id string) *Participants {
	var e Participants
	if d.db.QueryRow(`SELECT id, tournament_id, name, email, seed, status, created_at FROM participants WHERE id=?`, id).Scan(&e.ID, &e.TournamentId, &e.Name, &e.Email, &e.Seed, &e.Status, &e.CreatedAt) != nil { return nil }
	return &e
}

func (d *DB) ListParticipants() []Participants {
	rows, _ := d.db.Query(`SELECT id, tournament_id, name, email, seed, status, created_at FROM participants ORDER BY created_at DESC`)
	if rows == nil { return nil }; defer rows.Close()
	var o []Participants
	for rows.Next() { var e Participants; rows.Scan(&e.ID, &e.TournamentId, &e.Name, &e.Email, &e.Seed, &e.Status, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) UpdateParticipants(e *Participants) error {
	_, err := d.db.Exec(`UPDATE participants SET tournament_id=?, name=?, email=?, seed=?, status=? WHERE id=?`, e.TournamentId, e.Name, e.Email, e.Seed, e.Status, e.ID)
	return err
}

func (d *DB) DeleteParticipants(id string) error {
	_, err := d.db.Exec(`DELETE FROM participants WHERE id=?`, id)
	return err
}

func (d *DB) CountParticipants() int {
	var n int; d.db.QueryRow(`SELECT COUNT(*) FROM participants`).Scan(&n); return n
}

func (d *DB) SearchParticipants(q string, filters map[string]string) []Participants {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (tournament_id LIKE ? OR name LIKE ? OR email LIKE ?)"
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["status"]; ok && v != "" { where += " AND status=?"; args = append(args, v) }
	rows, _ := d.db.Query(`SELECT id, tournament_id, name, email, seed, status, created_at FROM participants WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil { return nil }; defer rows.Close()
	var o []Participants
	for rows.Next() { var e Participants; rows.Scan(&e.ID, &e.TournamentId, &e.Name, &e.Email, &e.Seed, &e.Status, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) CreateMatches(e *Matches) error {
	e.ID = genID(); e.CreatedAt = now()
	_, err := d.db.Exec(`INSERT INTO matches(id, tournament_id, round, match_number, player1, player2, score1, score2, winner, status, created_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, e.TournamentId, e.Round, e.MatchNumber, e.Player1, e.Player2, e.Score1, e.Score2, e.Winner, e.Status, e.CreatedAt)
	return err
}

func (d *DB) GetMatches(id string) *Matches {
	var e Matches
	if d.db.QueryRow(`SELECT id, tournament_id, round, match_number, player1, player2, score1, score2, winner, status, created_at FROM matches WHERE id=?`, id).Scan(&e.ID, &e.TournamentId, &e.Round, &e.MatchNumber, &e.Player1, &e.Player2, &e.Score1, &e.Score2, &e.Winner, &e.Status, &e.CreatedAt) != nil { return nil }
	return &e
}

func (d *DB) ListMatches() []Matches {
	rows, _ := d.db.Query(`SELECT id, tournament_id, round, match_number, player1, player2, score1, score2, winner, status, created_at FROM matches ORDER BY created_at DESC`)
	if rows == nil { return nil }; defer rows.Close()
	var o []Matches
	for rows.Next() { var e Matches; rows.Scan(&e.ID, &e.TournamentId, &e.Round, &e.MatchNumber, &e.Player1, &e.Player2, &e.Score1, &e.Score2, &e.Winner, &e.Status, &e.CreatedAt); o = append(o, e) }
	return o
}

func (d *DB) UpdateMatches(e *Matches) error {
	_, err := d.db.Exec(`UPDATE matches SET tournament_id=?, round=?, match_number=?, player1=?, player2=?, score1=?, score2=?, winner=?, status=? WHERE id=?`, e.TournamentId, e.Round, e.MatchNumber, e.Player1, e.Player2, e.Score1, e.Score2, e.Winner, e.Status, e.ID)
	return err
}

func (d *DB) DeleteMatches(id string) error {
	_, err := d.db.Exec(`DELETE FROM matches WHERE id=?`, id)
	return err
}

func (d *DB) CountMatches() int {
	var n int; d.db.QueryRow(`SELECT COUNT(*) FROM matches`).Scan(&n); return n
}

func (d *DB) SearchMatches(q string, filters map[string]string) []Matches {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (tournament_id LIKE ? OR player1 LIKE ? OR player2 LIKE ? OR score1 LIKE ? OR score2 LIKE ? OR winner LIKE ?)"
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
		args = append(args, "%"+q+"%")
	}
	if v, ok := filters["status"]; ok && v != "" { where += " AND status=?"; args = append(args, v) }
	rows, _ := d.db.Query(`SELECT id, tournament_id, round, match_number, player1, player2, score1, score2, winner, status, created_at FROM matches WHERE `+where+` ORDER BY created_at DESC`, args...)
	if rows == nil { return nil }; defer rows.Close()
	var o []Matches
	for rows.Next() { var e Matches; rows.Scan(&e.ID, &e.TournamentId, &e.Round, &e.MatchNumber, &e.Player1, &e.Player2, &e.Score1, &e.Score2, &e.Winner, &e.Status, &e.CreatedAt); o = append(o, e) }
	return o
}

// GetExtras returns the JSON extras blob for a record. Returns "{}" if none.
func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(`SELECT data FROM extras WHERE resource=? AND record_id=?`, resource, recordID).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

// SetExtras stores the JSON extras blob for a record.
func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?) ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`, resource, recordID, data)
	return err
}

// DeleteExtras removes extras when a record is deleted.
func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(`DELETE FROM extras WHERE resource=? AND record_id=?`, resource, recordID)
	return err
}

// AllExtras returns all extras for a resource type as a map of record_id → JSON string.
func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(`SELECT record_id, data FROM extras WHERE resource=?`, resource)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
