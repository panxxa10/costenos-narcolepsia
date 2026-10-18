package hospital

import "fmt"

// Orderly es un camillero: recoge al paciente dormido y lo lleva a la cama,
// pero no diagnostica. Implementa Attender.
type Orderly struct {
	Person
	transfers int
}

// NewOrderly crea un camillero.
func NewOrderly(id, name string, age int) *Orderly {
	return &Orderly{Person: newPerson(id, name, age)}
}

// Transfers devuelve cuántos pacientes ha llevado a una cama.
func (o *Orderly) Transfers() int { return o.transfers }

// Attend cumple la interfaz Attender: registra el episodio y, si el
// paciente ya tiene sala, cuenta el traslado. No toca al doctor asignado.
func (o *Orderly) Attend(p *Patient, location string) (EpisodeRecord, error) {
	if p == nil {
		return EpisodeRecord{}, ErrNilPatient
	}
	if p.state == Awake {
		return EpisodeRecord{}, fmt.Errorf("%s: %w", p.Name(), ErrPatientAwake)
	}
	if p.room != nil {
		o.transfers++
	}
	return newEpisodeRecord(p, o, location), nil
}
