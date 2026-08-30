package domain

import (
	"cmp"
	"slices"
)

// DoctorSimulationSlot tracks a doctor's projected availability time during queue simulation.
type DoctorSimulationSlot struct {
	Doctor            *Doctor
	NextAvailableTime int
}

// CalculateEstimatedWaitingTime computes the estimated waiting time in minutes
// for a patient given the list of online doctors and their 1-based queue position.
//
// Scheduling Algorithm: Deterministic Greedy Earliest-Available-First Dispatch.
// 1. Initial next availability is derived from the doctor's active consultation remaining time.
// 2. Preceding waiting patients (position 1 to N-1) are sequentially dispatched to the
//    doctor who becomes free earliest (tie-breaking with fastest average consultation time).
// 3. The target patient's estimated wait time is the earliest available slot among all doctors.
func CalculateEstimatedWaitingTime(doctors []*Doctor, positionInQueue int) (int, error) {
	if len(doctors) == 0 {
		return 0, ErrEmptyDoctors
	}

	if positionInQueue <= 0 {
		return 0, ErrInvalidPosition
	}

	// Initialize simulation slots
	slots := make([]*DoctorSimulationSlot, len(doctors))
	for i, doc := range doctors {
		if doc == nil {
			return 0, ErrNilDoctor
		}
		if doc.AvgConsultationTime <= 0 {
			return 0, ErrInvalidConsultationTime
		}

		slots[i] = &DoctorSimulationSlot{
			Doctor:            doc,
			NextAvailableTime: doc.CurrentSession.RemainingTime(doc.AvgConsultationTime),
		}
	}

	patientsAhead := positionInQueue - 1

	// Greedy Dispatch: Allocate each waiting patient ahead to the doctor free earliest
	for range patientsAhead {
		slices.SortFunc(slots, func(a, b *DoctorSimulationSlot) int {
			if n := cmp.Compare(a.NextAvailableTime, b.NextAvailableTime); n != 0 {
				return n
			}
			return cmp.Compare(a.Doctor.AvgConsultationTime, b.Doctor.AvgConsultationTime)
		})

		slots[0].NextAvailableTime += slots[0].Doctor.AvgConsultationTime
	}

	// Target patient's wait time is the earliest time when any doctor is next free
	slices.SortFunc(slots, func(a, b *DoctorSimulationSlot) int {
		return cmp.Compare(a.NextAvailableTime, b.NextAvailableTime)
	})

	return slots[0].NextAvailableTime, nil
}
