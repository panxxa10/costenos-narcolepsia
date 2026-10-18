// Package web expone el hospital como una API REST con net/http y sirve
// un frontend HTML/JS embebido. El paquete hospital no sabe que esto existe.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"costenos-narcolepsia/hospital"
)

//go:embed static
var staticFiles embed.FS

// Server atiende las peticiones HTTP sobre un Hospital.
type Server struct {
	h   *hospital.Hospital
	mux *http.ServeMux
}

// NewServer crea el servidor con sus rutas:
//
//	GET  /api/state   foto del hospital en JSON
//	POST /api/attack  {"patientId", "location"}
//	POST /api/wake    {"patientId"}
//	GET  /            frontend
func NewServer(h *hospital.Hospital) *Server {
	s := &Server{h: h, mux: http.NewServeMux()}
	static, _ := fs.Sub(staticFiles, "static") // "static" existe por el embed, no falla
	s.mux.Handle("/", http.FileServer(http.FS(static)))
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/attack", s.handleAttack)
	s.mux.HandleFunc("/api/wake", s.handleWake)
	return s
}

// ServeHTTP hace que Server cumpla http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe levanta el servidor en addr (por ejemplo ":8080").
func ListenAndServe(addr string, h *hospital.Hospital) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           NewServer(h),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return srv.ListenAndServe()
}

type attackRequest struct {
	PatientID string `json:"patientId"`
	Location  string `json:"location"`
}

type wakeRequest struct {
	PatientID string `json:"patientId"`
}

// actionResponse es la respuesta de /api/attack y /api/wake.
type actionResponse struct {
	OK       bool     `json:"ok"`
	Message  string   `json:"message"`
	Warning  string   `json:"warning,omitempty"`
	Episodes []string `json:"episodes,omitempty"`
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, errors.New("usa GET"))
		return
	}
	writeJSON(w, http.StatusOK, s.h.Snapshot())
}

func (s *Server) handleAttack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("usa POST"))
		return
	}
	var req attackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("JSON inválido"))
		return
	}
	if req.PatientID == "" || req.Location == "" {
		writeError(w, http.StatusBadRequest, errors.New("faltan patientId o location"))
		return
	}

	rec, err := s.h.HandleSleepAttack(req.PatientID, req.Location)
	switch {
	case errors.Is(err, hospital.ErrNoRoomAvailable):
		// No es un fallo de la petición: el episodio sí quedó registrado.
		writeJSON(w, http.StatusOK, actionResponse{
			OK: true, Message: rec.Summary(), Warning: err.Error(),
		})
	case err != nil:
		writeError(w, statusFor(err), err)
	default:
		writeJSON(w, http.StatusOK, actionResponse{OK: true, Message: rec.Summary()})
	}
}

func (s *Server) handleWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("usa POST"))
		return
	}
	var req wakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PatientID == "" {
		writeError(w, http.StatusBadRequest, errors.New("JSON inválido o falta patientId"))
		return
	}
	if err := s.h.WakePatient(req.PatientID); err != nil {
		writeError(w, statusFor(err), err)
		return
	}
	resp := actionResponse{OK: true, Message: req.PatientID + " se despertó"}
	for _, rec := range s.h.RelocateHallwayPatients() {
		resp.Episodes = append(resp.Episodes, rec.Summary())
	}
	writeJSON(w, http.StatusOK, resp)
}

// statusFor traduce los errores del hospital a códigos HTTP.
func statusFor(err error) int {
	switch {
	case errors.Is(err, hospital.ErrPatientNotFound):
		return http.StatusNotFound
	case errors.Is(err, hospital.ErrAlreadyAsleep), errors.Is(err, hospital.ErrAlreadyAwake):
		return http.StatusConflict
	case errors.Is(err, hospital.ErrNoStaff):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // si el cliente se fue, no hay a quién avisarle
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, actionResponse{OK: false, Message: err.Error()})
}
