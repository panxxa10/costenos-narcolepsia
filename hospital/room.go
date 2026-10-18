package hospital

import (
	"fmt"
	"slices"
)

// RoomState indica si una sala tiene cupo o no.
type RoomState int

// Estados posibles de una sala.
const (
	Available RoomState = iota
	Occupied
)

// String devuelve el estado de la sala en texto legible.
func (s RoomState) String() string {
	switch s {
	case Available:
		return "disponible"
	case Occupied:
		return "ocupada"
	default:
		return fmt.Sprintf("RoomState(%d)", int(s))
	}
}

// Room es una sala con camas para acostar pacientes dormidos.
type Room struct {
	number    int
	capacity  int
	state     RoomState
	occupants []*Patient
}

// NewRoom crea una sala vacía. El número debe ser positivo y la capacidad
// al menos 1; si no, devuelve ErrInvalidRoom.
func NewRoom(number, capacity int) (*Room, error) {
	if number <= 0 || capacity <= 0 {
		return nil, fmt.Errorf("sala %d con capacidad %d: %w", number, capacity, ErrInvalidRoom)
	}
	return &Room{number: number, capacity: capacity, state: Available}, nil
}

// Number devuelve el número de la sala.
func (r *Room) Number() int { return r.number }

// Capacity devuelve cuántos pacientes caben en la sala.
func (r *Room) Capacity() int { return r.capacity }

// State devuelve si la sala está disponible u ocupada.
func (r *Room) State() RoomState { return r.state }

// Occupants devuelve una copia de la lista de pacientes en la sala.
func (r *Room) Occupants() []*Patient { return slices.Clone(r.occupants) }

// IsAvailable dice si todavía cabe alguien en la sala.
func (r *Room) IsAvailable() bool { return r.state == Available }

// Occupy mete al paciente en la sala. Si con él se llena, pasa a Occupied.
func (r *Room) Occupy(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	if !r.IsAvailable() {
		return fmt.Errorf("sala %d: %w", r.number, ErrRoomFull)
	}
	r.occupants = append(r.occupants, p)
	if len(r.occupants) == r.capacity {
		r.state = Occupied
	}
	return nil
}

// Release saca al paciente de la sala y la deja disponible otra vez.
func (r *Room) Release(p *Patient) error {
	i := slices.Index(r.occupants, p)
	if i < 0 {
		return fmt.Errorf("sala %d: %w", r.number, ErrNotInRoom)
	}
	r.occupants = slices.Delete(r.occupants, i, i+1)
	r.state = Available
	return nil
}
