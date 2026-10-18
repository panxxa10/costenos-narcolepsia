package hospital

import (
	"fmt"
	"slices"
)

// Doctor es un médico del hospital. Diagnostica pacientes y atiende
// emergencias de sueño. Implementa Attender.
type Doctor struct {
	Person
	specialty string
	patients  []*Patient
	episodes  []EpisodeRecord
}

// NewDoctor crea un doctor sin pacientes ni episodios.
func NewDoctor(id, name string, age int, specialty string) *Doctor {
	return &Doctor{Person: newPerson(id, name, age), specialty: specialty}
}

// Specialty devuelve la especialidad del doctor.
func (d *Doctor) Specialty() string { return d.specialty }

// Patients devuelve una copia de los pacientes a su cargo.
func (d *Doctor) Patients() []*Patient { return slices.Clone(d.patients) }

// DiagnosePatient deja al paciente a cargo de este doctor.
// Falla si el paciente ya tiene un doctor asignado.
func (d *Doctor) DiagnosePatient(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	if p.assignedDoctor != nil {
		return fmt.Errorf("%s ya está a cargo de %s: %w", p.Name(), p.assignedDoctor.Name(), ErrAlreadyHasDoctor)
	}
	p.assignedDoctor = d
	d.patients = append(d.patients, p)
	return nil
}

// AttendSleepEmergency revisa a un paciente dormido. Si nadie lo tenía
// a cargo, el doctor se lo queda (lo diagnostica).
func (d *Doctor) AttendSleepEmergency(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	if p.state == Awake {
		return fmt.Errorf("%s: %w", p.Name(), ErrPatientAwake)
	}
	if p.assignedDoctor == nil {
		return d.DiagnosePatient(p)
	}
	return nil
}

// Attend cumple la interfaz Attender: atiende la emergencia y guarda el
// episodio en la lista propia del doctor (por eso MyEpisodes no necesita
// recibir el hospital).
func (d *Doctor) Attend(p *Patient, location string) (EpisodeRecord, error) {
	if err := d.AttendSleepEmergency(p); err != nil {
		return EpisodeRecord{}, err
	}
	rec := newEpisodeRecord(p, d, location)
	d.episodes = append(d.episodes, rec)
	return rec, nil
}

// MyEpisodes devuelve una copia de los episodios que atendió este doctor.
func (d *Doctor) MyEpisodes() []EpisodeRecord { return slices.Clone(d.episodes) }
