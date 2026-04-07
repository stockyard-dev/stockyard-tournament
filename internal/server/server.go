package server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-tournament/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/tournaments", s.listTournaments)
	s.mux.HandleFunc("POST /api/tournaments", s.createTournaments)
	s.mux.HandleFunc("GET /api/tournaments/export.csv", s.exportTournaments)
	s.mux.HandleFunc("GET /api/tournaments/{id}", s.getTournaments)
	s.mux.HandleFunc("PUT /api/tournaments/{id}", s.updateTournaments)
	s.mux.HandleFunc("DELETE /api/tournaments/{id}", s.delTournaments)
	s.mux.HandleFunc("GET /api/participants", s.listParticipants)
	s.mux.HandleFunc("POST /api/participants", s.createParticipants)
	s.mux.HandleFunc("GET /api/participants/export.csv", s.exportParticipants)
	s.mux.HandleFunc("GET /api/participants/{id}", s.getParticipants)
	s.mux.HandleFunc("PUT /api/participants/{id}", s.updateParticipants)
	s.mux.HandleFunc("DELETE /api/participants/{id}", s.delParticipants)
	s.mux.HandleFunc("GET /api/matches", s.listMatches)
	s.mux.HandleFunc("POST /api/matches", s.createMatches)
	s.mux.HandleFunc("GET /api/matches/export.csv", s.exportMatches)
	s.mux.HandleFunc("GET /api/matches/{id}", s.getMatches)
	s.mux.HandleFunc("PUT /api/matches/{id}", s.updateMatches)
	s.mux.HandleFunc("DELETE /api/matches/{id}", s.delMatches)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", s.tierHandler)
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(c); json.NewEncoder(w).Encode(v) }
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) { if r.URL.Path != "/" { http.NotFound(w, r); return }; http.Redirect(w, r, "/ui", 302) }
func oe[T any](s []T) []T { if s == nil { return []T{} }; return s }
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

func (s *Server) listTournaments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("format"); v != "" { filters["format"] = v }
	if v := r.URL.Query().Get("status"); v != "" { filters["status"] = v }
	if q != "" || len(filters) > 0 { wj(w, 200, map[string]any{"tournaments": oe(s.db.SearchTournaments(q, filters))}); return }
	wj(w, 200, map[string]any{"tournaments": oe(s.db.ListTournaments())})
}

func (s *Server) createTournaments(w http.ResponseWriter, r *http.Request) {
	if s.limits.Tier == "none" { we(w, 402, "No license key. Start a 14-day trial at https://stockyard.dev/for/"); return }
	if s.limits.TrialExpired { we(w, 402, "Trial expired. Subscribe at https://stockyard.dev/pricing/"); return }
	var e store.Tournaments
	json.NewDecoder(r.Body).Decode(&e)
	if e.Name == "" { we(w, 400, "name required"); return }
	s.db.CreateTournaments(&e)
	wj(w, 201, s.db.GetTournaments(e.ID))
}

func (s *Server) getTournaments(w http.ResponseWriter, r *http.Request) {
	e := s.db.GetTournaments(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, 200, e)
}

func (s *Server) updateTournaments(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetTournaments(r.PathValue("id"))
	if existing == nil { we(w, 404, "not found"); return }
	var patch store.Tournaments
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID; patch.CreatedAt = existing.CreatedAt
	if patch.Name == "" { patch.Name = existing.Name }
	if patch.Game == "" { patch.Game = existing.Game }
	if patch.Format == "" { patch.Format = existing.Format }
	if patch.Date == "" { patch.Date = existing.Date }
	if patch.Location == "" { patch.Location = existing.Location }
	if patch.Status == "" { patch.Status = existing.Status }
	if patch.Notes == "" { patch.Notes = existing.Notes }
	s.db.UpdateTournaments(&patch)
	wj(w, 200, s.db.GetTournaments(patch.ID))
}

func (s *Server) delTournaments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id"); s.db.DeleteTournaments(id); s.db.DeleteExtras("tournaments", id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) exportTournaments(w http.ResponseWriter, r *http.Request) {
	items := s.db.ListTournaments()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=tournaments.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "name", "game", "format", "date", "location", "max_participants", "status", "notes", "created_at"})
	for _, e := range items {
		cw.Write([]string{e.ID, fmt.Sprintf("%v", e.Name), fmt.Sprintf("%v", e.Game), fmt.Sprintf("%v", e.Format), fmt.Sprintf("%v", e.Date), fmt.Sprintf("%v", e.Location), fmt.Sprintf("%v", e.MaxParticipants), fmt.Sprintf("%v", e.Status), fmt.Sprintf("%v", e.Notes), e.CreatedAt})
	}
	cw.Flush()
}

func (s *Server) listParticipants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("status"); v != "" { filters["status"] = v }
	if q != "" || len(filters) > 0 { wj(w, 200, map[string]any{"participants": oe(s.db.SearchParticipants(q, filters))}); return }
	wj(w, 200, map[string]any{"participants": oe(s.db.ListParticipants())})
}

func (s *Server) createParticipants(w http.ResponseWriter, r *http.Request) {
	var e store.Participants
	json.NewDecoder(r.Body).Decode(&e)
	if e.TournamentId == "" { we(w, 400, "tournament_id required"); return }
	if e.Name == "" { we(w, 400, "name required"); return }
	s.db.CreateParticipants(&e)
	wj(w, 201, s.db.GetParticipants(e.ID))
}

func (s *Server) getParticipants(w http.ResponseWriter, r *http.Request) {
	e := s.db.GetParticipants(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, 200, e)
}

func (s *Server) updateParticipants(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetParticipants(r.PathValue("id"))
	if existing == nil { we(w, 404, "not found"); return }
	var patch store.Participants
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID; patch.CreatedAt = existing.CreatedAt
	if patch.TournamentId == "" { patch.TournamentId = existing.TournamentId }
	if patch.Name == "" { patch.Name = existing.Name }
	if patch.Email == "" { patch.Email = existing.Email }
	if patch.Status == "" { patch.Status = existing.Status }
	s.db.UpdateParticipants(&patch)
	wj(w, 200, s.db.GetParticipants(patch.ID))
}

func (s *Server) delParticipants(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id"); s.db.DeleteParticipants(id); s.db.DeleteExtras("participants", id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) exportParticipants(w http.ResponseWriter, r *http.Request) {
	items := s.db.ListParticipants()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=participants.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "tournament_id", "name", "email", "seed", "status", "created_at"})
	for _, e := range items {
		cw.Write([]string{e.ID, fmt.Sprintf("%v", e.TournamentId), fmt.Sprintf("%v", e.Name), fmt.Sprintf("%v", e.Email), fmt.Sprintf("%v", e.Seed), fmt.Sprintf("%v", e.Status), e.CreatedAt})
	}
	cw.Flush()
}

func (s *Server) listMatches(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("status"); v != "" { filters["status"] = v }
	if q != "" || len(filters) > 0 { wj(w, 200, map[string]any{"matches": oe(s.db.SearchMatches(q, filters))}); return }
	wj(w, 200, map[string]any{"matches": oe(s.db.ListMatches())})
}

func (s *Server) createMatches(w http.ResponseWriter, r *http.Request) {
	var e store.Matches
	json.NewDecoder(r.Body).Decode(&e)
	if e.TournamentId == "" { we(w, 400, "tournament_id required"); return }
	s.db.CreateMatches(&e)
	wj(w, 201, s.db.GetMatches(e.ID))
}

func (s *Server) getMatches(w http.ResponseWriter, r *http.Request) {
	e := s.db.GetMatches(r.PathValue("id"))
	if e == nil { we(w, 404, "not found"); return }
	wj(w, 200, e)
}

func (s *Server) updateMatches(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetMatches(r.PathValue("id"))
	if existing == nil { we(w, 404, "not found"); return }
	var patch store.Matches
	json.NewDecoder(r.Body).Decode(&patch)
	patch.ID = existing.ID; patch.CreatedAt = existing.CreatedAt
	if patch.TournamentId == "" { patch.TournamentId = existing.TournamentId }
	if patch.Player1 == "" { patch.Player1 = existing.Player1 }
	if patch.Player2 == "" { patch.Player2 = existing.Player2 }
	if patch.Score1 == "" { patch.Score1 = existing.Score1 }
	if patch.Score2 == "" { patch.Score2 = existing.Score2 }
	if patch.Winner == "" { patch.Winner = existing.Winner }
	if patch.Status == "" { patch.Status = existing.Status }
	s.db.UpdateMatches(&patch)
	wj(w, 200, s.db.GetMatches(patch.ID))
}

func (s *Server) delMatches(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id"); s.db.DeleteMatches(id); s.db.DeleteExtras("matches", id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) exportMatches(w http.ResponseWriter, r *http.Request) {
	items := s.db.ListMatches()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=matches.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "tournament_id", "round", "match_number", "player1", "player2", "score1", "score2", "winner", "status", "created_at"})
	for _, e := range items {
		cw.Write([]string{e.ID, fmt.Sprintf("%v", e.TournamentId), fmt.Sprintf("%v", e.Round), fmt.Sprintf("%v", e.MatchNumber), fmt.Sprintf("%v", e.Player1), fmt.Sprintf("%v", e.Player2), fmt.Sprintf("%v", e.Score1), fmt.Sprintf("%v", e.Score2), fmt.Sprintf("%v", e.Winner), fmt.Sprintf("%v", e.Status), e.CreatedAt})
	}
	cw.Flush()
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{}
	m["tournaments_total"] = s.db.CountTournaments()
	m["participants_total"] = s.db.CountParticipants()
	m["matches_total"] = s.db.CountMatches()
	wj(w, 200, m)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{"status": "ok", "service": "tournament"}
	m["tournaments"] = s.db.CountTournaments()
	m["participants"] = s.db.CountParticipants()
	m["matches"] = s.db.CountMatches()
	wj(w, 200, m)
}

// loadPersonalConfig reads config.json from the data directory.
func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// listExtras returns all extras for a resource type as {record_id: {...fields...}}
func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	wj(w, 200, out)
}

// getExtras returns the extras blob for a single record.
func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

// putExtras stores the extras blob for a single record.
func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		we(w, 400, "read body")
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		we(w, 500, "save failed")
		return
	}
	wj(w, 200, map[string]string{"ok": "saved"})
}
