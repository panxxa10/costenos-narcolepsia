package hospital

import (
	"fmt"
	"sync/atomic"
	"time"
)

// episodeSeq genera IDs únicos de episodio. Es atómico porque los
// Attender crean episodios y pueden hacerlo desde varias goroutines.
var episodeSeq atomic.Int64

// EpisodeRecord es el registro de un ataque de sueño: quién se durmió,
// dónde, quién lo atendió y a qué sala fue a parar (0 si a ninguna).
type EpisodeRecord struct {
	id        string
	at        time.Time
	patient   *Patient
	attending Attender
	location  string
	room      int
}

// newEpisodeRecord crea el registro con la hora actual. Si el paciente ya
// tiene sala, la anota; si no, el episodio queda "sin sala".
func newEpisodeRecord(p *Patient, a Attender, location string) EpisodeRecord {
	room := 0
	if p.room != nil {
		room = p.room.Number()
	}
	return EpisodeRecord{
		id:        fmt.Sprintf("EP-%03d", episodeSeq.Add(1)),
		at:        time.Now(),
		patient:   p,
		attending: a,
		location:  location,
		room:      room,
	}
}

// ID devuelve el identificador del episodio.
func (e EpisodeRecord) ID() string { return e.id }

// At devuelve la fecha y hora del episodio.
func (e EpisodeRecord) At() time.Time { return e.at }

// Patient devuelve el paciente que se durmió.
func (e EpisodeRecord) Patient() *Patient { return e.patient }

// Attending devuelve quién lo atendió (doctor, camillero, ...).
func (e EpisodeRecord) Attending() Attender { return e.attending }

// Location devuelve dónde se quedó dormido el paciente.
func (e EpisodeRecord) Location() string { return e.location }

// RoomNumber devuelve la sala asignada, o 0 si se quedó sin sala.
func (e EpisodeRecord) RoomNumber() int { return e.room }

// HasRoom dice si en este episodio se le consiguió sala.
func (e EpisodeRecord) HasRoom() bool { return e.room != 0 }

// Summary devuelve una línea con el resumen del episodio.
func (e EpisodeRecord) Summary() string {
	where := "sin sala, se quedó en el pasillo"
	if e.HasRoom() {
		where = fmt.Sprintf("sala %d", e.room)
	}
	return fmt.Sprintf("%s [%s] %s (%s) | lugar: %s | atendió: %s | %s",
		e.id, e.at.Format("2006-01-02 15:04:05"), e.patient.Name(), e.patient.ID(),
		e.location, e.attending.Name(), where)
}
