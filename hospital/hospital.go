// Package hospital modela el Hospital de los Costeños con Narcolepsia:
// pacientes que se duermen en cualquier parte, doctores, camilleros, salas
// y el historial de episodios de sueño. Solo contiene modelo y lógica;
// nada de presentación.
package hospital

import (
	"errors"
	"fmt"
	"sync"
)

// Errores centinela del paquete. Se comparan con errors.Is porque casi
// siempre llegan envueltos con contexto (fmt.Errorf con %w).
var (
	ErrNoRoomAvailable  = errors.New("no hay sala disponible")
	ErrPatientNotFound  = errors.New("paciente no encontrado")
	ErrDuplicateID      = errors.New("ya existe alguien con ese ID")
	ErrDuplicateRoom    = errors.New("ya existe una sala con ese número")
	ErrInvalidRoom      = errors.New("sala inválida")
	ErrNoStaff          = errors.New("no hay personal para despachar")
	ErrNilPatient       = errors.New("paciente nil")
	ErrAlreadyAsleep    = errors.New("el paciente ya está dormido")
	ErrAlreadyAwake     = errors.New("el paciente ya está despierto")
	ErrAlreadyInBed     = errors.New("el paciente ya tiene cama")
	ErrPatientAwake     = errors.New("el paciente está despierto")
	ErrAlreadyHasDoctor = errors.New("el paciente ya tiene doctor")
	ErrRoomFull         = errors.New("la sala está llena")
	ErrNotInRoom        = errors.New("el paciente no está en esa sala")
	ErrInvalidEpisode   = errors.New("episodio inválido")
)

// Attender es cualquiera que pueda atender a un paciente dormido.
// La implementan Doctor y Orderly; el Hospital los despacha sin saber
// cuál es el tipo concreto.
type Attender interface {
	ID() string
	Name() string
	Attend(p *Patient, location string) (EpisodeRecord, error)
}

// Hospital es la raíz del modelo. Todo su estado está protegido por mu,
// así que sus métodos públicos se pueden llamar desde varias goroutines.
// Regla para evitar deadlocks: los métodos públicos toman el lock y
// delegan en helpers terminados en "Locked", que nunca lo toman.
type Hospital struct {
	mu        sync.Mutex
	name      string
	doctors   []*Doctor
	patients  []*Patient
	rooms     []*Room
	staff     []Attender
	history   []EpisodeRecord
	nextStaff int
}

// NewHospital crea un hospital vacío.
func NewHospital(name string) *Hospital {
	return &Hospital{name: name}
}

// Name devuelve el nombre del hospital.
func (h *Hospital) Name() string { return h.name }

// HireDoctor contrata un doctor: queda en la lista de doctores y en el
// personal que se despacha por turnos.
func (h *Hospital) HireDoctor(d *Doctor) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if d == nil {
		return errors.New("doctor nil")
	}
	if err := h.hireLocked(d); err != nil {
		return err
	}
	h.doctors = append(h.doctors, d)
	return nil
}

// HireStaff contrata a cualquier Attender que no sea doctor (camilleros,
// enfermeras, ...). Entra a la rotación de turnos.
func (h *Hospital) HireStaff(a Attender) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if a == nil {
		return errors.New("personal nil")
	}
	return h.hireLocked(a)
}

// hireLocked agrega a al personal si su ID no está repetido.
func (h *Hospital) hireLocked(a Attender) error {
	for _, s := range h.staff {
		if s.ID() == a.ID() {
			return fmt.Errorf("personal %s: %w", a.ID(), ErrDuplicateID)
		}
	}
	h.staff = append(h.staff, a)
	return nil
}

// AddRoom registra una sala nueva en el hospital.
func (h *Hospital) AddRoom(r *Room) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r == nil {
		return ErrInvalidRoom
	}
	for _, existing := range h.rooms {
		if existing.Number() == r.Number() {
			return fmt.Errorf("sala %d: %w", r.Number(), ErrDuplicateRoom)
		}
	}
	h.rooms = append(h.rooms, r)
	return nil
}

// AdmitPatient admite un paciente. Falla si ya hay otro con el mismo ID.
func (h *Hospital) AdmitPatient(p *Patient) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if p == nil {
		return ErrNilPatient
	}
	if _, err := h.findPatientLocked(p.ID()); err == nil {
		return fmt.Errorf("paciente %s: %w", p.ID(), ErrDuplicateID)
	}
	h.patients = append(h.patients, p)
	return nil
}

// FindPatient busca un paciente admitido por su ID.
func (h *Hospital) FindPatient(id string) (*Patient, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.findPatientLocked(id)
}

func (h *Hospital) findPatientLocked(id string) (*Patient, error) {
	for _, p := range h.patients {
		if p.ID() == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%q: %w", id, ErrPatientNotFound)
}

// Doctors devuelve una copia de la lista de doctores contratados.
func (h *Hospital) Doctors() []*Doctor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Doctor(nil), h.doctors...)
}

// DispatchAttender devuelve el siguiente miembro del personal por turnos
// (round-robin). No le importa si es doctor, camillero u otro Attender.
func (h *Hospital) DispatchAttender() (Attender, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dispatchLocked()
}

func (h *Hospital) dispatchLocked() (Attender, error) {
	if len(h.staff) == 0 {
		return nil, ErrNoStaff
	}
	a := h.staff[h.nextStaff%len(h.staff)]
	h.nextStaff++
	return a, nil
}

// RegisterEpisode agrega un episodio al historial. El paciente del
// episodio debe estar admitido en este hospital.
func (h *Hospital) RegisterEpisode(rec EpisodeRecord) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.registerEpisodeLocked(rec)
}

func (h *Hospital) registerEpisodeLocked(rec EpisodeRecord) error {
	if rec.patient == nil || rec.attending == nil {
		return ErrInvalidEpisode
	}
	if _, err := h.findPatientLocked(rec.patient.ID()); err != nil {
		return err
	}
	h.history = append(h.history, rec)
	return nil
}

// HandleSleepAttack es el flujo completo de un ataque de sueño:
//  1. el paciente queda AsleepInHallway en location,
//  2. se despacha un Attender por turnos,
//  3. se intenta asignar la primera sala libre,
//  4. el Attender atiende y el episodio se guarda en el historial.
//
// Si no hay sala, el episodio igual queda registrado (sin sala), el
// paciente sigue en el pasillo y se devuelve ErrNoRoomAvailable envuelto.
func (h *Hospital) HandleSleepAttack(patientID, location string) (EpisodeRecord, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	p, err := h.findPatientLocked(patientID)
	if err != nil {
		return EpisodeRecord{}, err
	}
	if p.state != Awake {
		return EpisodeRecord{}, fmt.Errorf("%s: %w", p.Name(), ErrAlreadyAsleep)
	}
	attender, err := h.dispatchLocked()
	if err != nil {
		return EpisodeRecord{}, err
	}

	p.SufferSleepAttack(location)
	_, roomErr := h.assignRoomLocked(p)

	rec, err := attender.Attend(p, location)
	if err != nil {
		return EpisodeRecord{}, err
	}
	if err := h.registerEpisodeLocked(rec); err != nil {
		return EpisodeRecord{}, err
	}
	return rec, roomErr
}

// AssignRoom le busca sala a un paciente que está dormido en el pasillo:
// toma la primera sala disponible, la ocupa y acuesta al paciente.
// Como es una reubicación, registra un episodio de seguimiento.
// Sin sala libre devuelve ErrNoRoomAvailable y el paciente sigue en el pasillo.
func (h *Hospital) AssignRoom(p *Patient) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if p == nil {
		return nil, ErrNilPatient
	}
	if _, err := h.findPatientLocked(p.ID()); err != nil {
		return nil, err
	}
	room, _, err := h.relocateLocked(p)
	return room, err
}

// assignRoomLocked busca la primera sala disponible y acuesta al paciente.
// Es el único lugar donde se ocupa una sala, siempre con el lock tomado,
// así dos goroutines nunca pueden quedarse con la misma.
func (h *Hospital) assignRoomLocked(p *Patient) (*Room, error) {
	switch p.state {
	case Awake:
		return nil, fmt.Errorf("%s: %w", p.Name(), ErrPatientAwake)
	case AsleepInBed:
		return nil, fmt.Errorf("%s: %w", p.Name(), ErrAlreadyInBed)
	}
	for _, r := range h.rooms {
		if !r.IsAvailable() {
			continue
		}
		if err := r.Occupy(p); err != nil {
			return nil, err
		}
		p.goToBed(r)
		return r, nil
	}
	return nil, fmt.Errorf("%s (%s) sigue dormido en %q: %w",
		p.Name(), p.ID(), p.currentLocation, ErrNoRoomAvailable)
}

// relocateLocked pasa a un paciente del pasillo a una sala y registra el
// episodio de seguimiento con el siguiente Attender de turno.
func (h *Hospital) relocateLocked(p *Patient) (*Room, EpisodeRecord, error) {
	if len(h.staff) == 0 {
		return nil, EpisodeRecord{}, ErrNoStaff
	}
	from := p.currentLocation
	room, err := h.assignRoomLocked(p)
	if err != nil {
		return nil, EpisodeRecord{}, err
	}
	attender, err := h.dispatchLocked()
	if err != nil {
		return room, EpisodeRecord{}, err
	}
	rec, err := attender.Attend(p, "seguimiento, reubicado desde "+from)
	if err != nil {
		return room, EpisodeRecord{}, err
	}
	if err := h.registerEpisodeLocked(rec); err != nil {
		return room, EpisodeRecord{}, err
	}
	return room, rec, nil
}

// WakePatient despierta al paciente con ese ID y libera su sala si tenía.
func (h *Hospital) WakePatient(patientID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	p, err := h.findPatientLocked(patientID)
	if err != nil {
		return err
	}
	return p.WakeUp()
}

// RelocateHallwayPatients reubica a todos los pacientes del pasillo que
// quepan en las salas libres (en orden de admisión) y devuelve los
// episodios de seguimiento creados.
func (h *Hospital) RelocateHallwayPatients() []EpisodeRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	var moved []EpisodeRecord
	for _, p := range h.patients {
		if p.state != AsleepInHallway {
			continue
		}
		_, rec, err := h.relocateLocked(p)
		if err != nil {
			break // no quedan salas (o no hay personal)
		}
		moved = append(moved, rec)
	}
	return moved
}

// History devuelve una copia de todos los episodios registrados.
func (h *Hospital) History() []EpisodeRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]EpisodeRecord(nil), h.history...)
}
