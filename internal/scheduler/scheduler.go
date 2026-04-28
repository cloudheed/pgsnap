// Package scheduler provides scheduled backup functionality.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Job represents a scheduled backup job.
type Job struct {
	Name     string
	Schedule Schedule
	Func     func(ctx context.Context) error
	LastRun  time.Time
	NextRun  time.Time
	Running  bool
	Errors   []error
}

// Schedule defines when a job should run.
type Schedule struct {
	// Interval between runs (e.g., 1*time.Hour for hourly)
	Interval time.Duration

	// Cron-like fields (optional, if Interval is 0)
	Hour   int // 0-23, -1 for any
	Minute int // 0-59, -1 for any
	Day    int // 1-31, -1 for any
}

// Hourly returns a schedule that runs every hour.
func Hourly() Schedule {
	return Schedule{Interval: time.Hour}
}

// Daily returns a schedule that runs daily at the specified hour.
func Daily(hour int) Schedule {
	return Schedule{Hour: hour, Minute: 0, Day: -1}
}

// Every returns a schedule that runs at the specified interval.
func Every(d time.Duration) Schedule {
	return Schedule{Interval: d}
}

// Scheduler manages scheduled jobs.
type Scheduler struct {
	jobs   map[string]*Job
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Scheduler.
func New() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		jobs:   make(map[string]*Job),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Add adds a job to the scheduler.
func (s *Scheduler) Add(name string, schedule Schedule, fn func(ctx context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job %q already exists", name)
	}

	job := &Job{
		Name:     name,
		Schedule: schedule,
		Func:     fn,
		NextRun:  calculateNextRun(schedule, time.Now()),
	}

	s.jobs[name] = job
	return nil
}

// Remove removes a job from the scheduler.
func (s *Scheduler) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; !exists {
		return fmt.Errorf("job %q not found", name)
	}

	delete(s.jobs, name)
	return nil
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go s.run()
}

// Stop stops the scheduler and waits for running jobs to finish.
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// Jobs returns a copy of all jobs.
func (s *Scheduler) Jobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		// Return a copy
		jobCopy := *j
		jobs = append(jobs, &jobCopy)
	}
	return jobs
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.checkAndRunJobs(now)
		}
	}
}

func (s *Scheduler) checkAndRunJobs(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		if job.Running {
			continue
		}

		if now.After(job.NextRun) || now.Equal(job.NextRun) {
			job.Running = true
			job.LastRun = now
			job.NextRun = calculateNextRun(job.Schedule, now)

			// Run job in goroutine
			go func(j *Job) {
				if err := j.Func(s.ctx); err != nil {
					s.mu.Lock()
					j.Errors = append(j.Errors, err)
					// Keep only last 10 errors
					if len(j.Errors) > 10 {
						j.Errors = j.Errors[len(j.Errors)-10:]
					}
					s.mu.Unlock()
				}

				s.mu.Lock()
				j.Running = false
				s.mu.Unlock()
			}(job)
		}
	}
}

func calculateNextRun(schedule Schedule, from time.Time) time.Time {
	if schedule.Interval > 0 {
		return from.Add(schedule.Interval)
	}

	// Cron-like scheduling
	next := from

	// Move to next minute
	next = next.Add(time.Minute)
	next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), 0, 0, next.Location())

	// Find next matching time
	for i := 0; i < 366*24*60; i++ { // Max 1 year of minutes
		if matchesSchedule(schedule, next) {
			return next
		}
		next = next.Add(time.Minute)
	}

	// Fallback: run in 24 hours
	return from.Add(24 * time.Hour)
}

func matchesSchedule(schedule Schedule, t time.Time) bool {
	if schedule.Hour >= 0 && t.Hour() != schedule.Hour {
		return false
	}
	if schedule.Minute >= 0 && t.Minute() != schedule.Minute {
		return false
	}
	if schedule.Day >= 0 && t.Day() != schedule.Day {
		return false
	}
	return true
}

// RunNow runs a job immediately.
func (s *Scheduler) RunNow(name string) error {
	s.mu.Lock()
	job, exists := s.jobs[name]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("job %q not found", name)
	}

	if job.Running {
		s.mu.Unlock()
		return fmt.Errorf("job %q is already running", name)
	}

	job.Running = true
	job.LastRun = time.Now()
	s.mu.Unlock()

	err := job.Func(s.ctx)

	s.mu.Lock()
	job.Running = false
	if err != nil {
		job.Errors = append(job.Errors, err)
	}
	s.mu.Unlock()

	return err
}
