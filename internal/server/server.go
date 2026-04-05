package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/stockyard-dev/stockyard-tournament/internal/store"
)

type Server struct {
	db     *store.DB
	mux    *http.ServeMux
	limits Limits
}

func New(db *store.DB, limits Limits) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits}
	s.mux.HandleFunc("GET /api/tournaments", s.listTournaments)
	s.mux.HandleFunc("POST /api/tournaments", s.createTournaments)
	s.mux.HandleFunc("GET /api/tournaments/{id}", s.getTournaments)
	s.mux.HandleFunc("PUT /api/tournaments/{id}", s.updateTournaments)
	s.mux.HandleFunc("DELETE /api/tournaments/{id}", s.delTournaments)
	s.mux.HandleFunc("GET /api/participants", s.listParticipants)
	s.mux.HandleFunc("POST /api/participants", s.createParticipants)
	s.mux.HandleFunc("GET /api/participants/{id}", s.getParticipants)
	s.mux.HandleFunc("PUT /api/participants/{id}", s.updateParticipants)
	s.mux.HandleFunc("DELETE /api/participants/{id}", s.delParticipants)
	s.mux.HandleFunc("GET /api/matches", s.listMatches)
	s.mux.HandleFunc("POST /api/matches", s.createMatches)
	s.mux.HandleFunc("GET /api/matches/{id}", s.getMatches)
	s.mux.HandleFunc("PUT /api/matches/{id}", s.updateMatches)
	s.mux.HandleFunc("DELETE /api/matches/{id}", s.delMatches)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/tournament/"})})
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
	if s.limits.MaxItems > 0 { if s.db.CountTournaments() >= s.limits.MaxItems { we(w, 402, "Free tier limit reached. Upgrade at https://stockyard.dev/tournament/"); return } }
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
	s.db.DeleteTournaments(r.PathValue("id"))
	wj(w, 200, map[string]string{"deleted": "ok"})
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
	s.db.DeleteParticipants(r.PathValue("id"))
	wj(w, 200, map[string]string{"deleted": "ok"})
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
	s.db.DeleteMatches(r.PathValue("id"))
	wj(w, 200, map[string]string{"deleted": "ok"})
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
