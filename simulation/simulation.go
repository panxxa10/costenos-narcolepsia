// Package simulation pone a cada paciente a dormirse y despertarse en su
// propia goroutine. Solo usa métodos públicos del Hospital, que están
// protegidos por su mutex.
package simulation

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"costenos-narcolepsia/hospital"
)

// locations son los sitios donde a un costeño le puede dar sueño.
var locations = []string{
	"la cafetería", "la fila de radiología", "el pasillo del segundo piso",
	"el ascensor", "la sala de espera", "el parqueadero", "la farmacia",
	"el baño del primer piso", "la escalera de emergencia",
}

// Stats resume lo que pasó en la simulación.
type Stats struct {
	Attacks     int64
	NoRoom      int64
	Wakes       int64
	Relocations int64
}

// counters son los contadores compartidos entre goroutines (atómicos).
type counters struct {
	attacks, noRoom, wakes, relocations atomic.Int64
}

// Run lanza una goroutine por paciente y espera a que todas terminen
// cuando se cumple la duración d (context con timeout).
func Run(h *hospital.Hospital, d time.Duration) Stats {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	var c counters
	var wg sync.WaitGroup
	for _, p := range h.Snapshot().Patients {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			patientLoop(ctx, h, id, &c)
		}(p.ID)
	}
	wg.Wait()

	return Stats{
		Attacks:     c.attacks.Load(),
		NoRoom:      c.noRoom.Load(),
		Wakes:       c.wakes.Load(),
		Relocations: c.relocations.Load(),
	}
}

// patientLoop es la vida de un paciente: despierto un rato, se duerme,
// duerme un rato, se despierta... hasta que se cancela el contexto.
func patientLoop(ctx context.Context, h *hospital.Hospital, id string, c *counters) {
	for {
		if !pause(ctx, 20, 120) {
			return
		}
		_, err := h.HandleSleepAttack(id, locations[rand.Intn(len(locations))])
		if err != nil && !errors.Is(err, hospital.ErrNoRoomAvailable) {
			continue // por ejemplo ya estaba dormido
		}
		c.attacks.Add(1)
		if err != nil {
			c.noRoom.Add(1)
		}

		if !pause(ctx, 50, 200) {
			return
		}
		if err := h.WakePatient(id); err != nil {
			continue
		}
		c.wakes.Add(1)
		// la cama que quedó libre se la dan al primero del pasillo
		c.relocations.Add(int64(len(h.RelocateHallwayPatients())))
	}
}

// pause duerme entre minMs y maxMs milisegundos. Devuelve false si el
// contexto se canceló mientras tanto.
func pause(ctx context.Context, minMs, maxMs int) bool {
	d := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
