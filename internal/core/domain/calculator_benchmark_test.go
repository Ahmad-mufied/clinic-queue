package domain_test

import (
	"fmt"
	"testing"

	"clinic-queue/internal/core/domain"
)

// helper to construct a deterministic slice of doctors with specified count and active consultation state
func makeBenchmarkDoctors(count int, withActiveSessions bool) []*domain.Doctor {
	doctors := make([]*domain.Doctor, count)
	// Base average consultation times cycling through realistic clinic durations (3, 4, 5, 6, 7 minutes)
	avgTimes := []int{3, 4, 5, 6, 7}

	for i := 0; i < count; i++ {
		avg := avgTimes[i%len(avgTimes)]
		doc := &domain.Doctor{
			ID:                  fmt.Sprintf("01919df4-8e3b-7412-a1f9-90b567c9e1%02d", i+1),
			Name:                fmt.Sprintf("Doctor %d", i+1),
			AvgConsultationTime: avg,
			IsOnline:            true,
		}

		if withActiveSessions && i%2 == 1 {
			// Every second doctor has an active patient consultation in progress
			doc.CurrentSession = &domain.ActiveConsultation{
				PatientName: fmt.Sprintf("Patient In Room %d", i+1),
				ElapsedTime: (i % avg) + 1, // Elapsed time < AvgConsultationTime
			}
		}

		doctors[i] = doc
	}
	return doctors
}

// BenchmarkCalculateEstimatedWaitingTime benchmarks the queue waiting time estimation algorithm
// across a matrix of doctor capacities (2, 5, 10) and queue depths (10, 50, 100, 500, 1000).
func BenchmarkCalculateEstimatedWaitingTime(b *testing.B) {
	doctorCounts := []int{2, 5, 10}
	patientSizes := []int{10, 50, 100, 500, 1000}

	b.Run("IdleDoctors", func(b *testing.B) {
		for _, numDocs := range doctorCounts {
			for _, numPatients := range patientSizes {
				name := fmt.Sprintf("Doctors=%d/Patients=%d", numDocs, numPatients)
				doctors := makeBenchmarkDoctors(numDocs, false)

				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						_, err := domain.CalculateEstimatedWaitingTime(doctors, numPatients)
						if err != nil {
							b.Fatalf("unexpected error during benchmark: %v", err)
						}
					}
				})
			}
		}
	})

	b.Run("ActiveSessions", func(b *testing.B) {
		for _, numDocs := range doctorCounts {
			for _, numPatients := range patientSizes {
				name := fmt.Sprintf("Doctors=%d/Patients=%d", numDocs, numPatients)
				doctors := makeBenchmarkDoctors(numDocs, true)

				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						_, err := domain.CalculateEstimatedWaitingTime(doctors, numPatients)
						if err != nil {
							b.Fatalf("unexpected error during benchmark: %v", err)
						}
					}
				})
			}
		}
	})
}
