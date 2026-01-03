package core

import (
	"log"
	"time"
)

type EmbeddingJobType int

const (
	CreateUserEmbedding EmbeddingJobType = iota
	UpdateUserEmbedding
	DeleteUserEmbedding
	CreateProjectEmbedding
	UpdateProjectEmbedding
	DeleteProjectEmbedding
)

type UserEmbedding struct {
	ID        string
	Bio       string
	Skills    []string
	Interests []string
}

type ProjectEmbedding struct {
	ID          string
	Title       string
	Description string
	Skills      []string
}

type EmbeddingJob struct {
	Type        EmbeddingJobType
	Attempts    int
	MaxAttempts int

	// User fields
	User UserEmbedding

	// Project fields
	Project ProjectEmbedding
}

type EmbeddingHub struct {
	Jobs       chan *EmbeddingJob
	Quit       chan bool
	WorkerPool int
}

func NewGRPCHub(queueSize, workers int) *EmbeddingHub {
	return &EmbeddingHub{
		Jobs:       make(chan *EmbeddingJob, queueSize),
		Quit:       make(chan bool),
		WorkerPool: workers,
	}
}

func (e *EmbeddingHub) Run() {
	for range e.WorkerPool {
		go e.Worker()
	}
	log.Println("[EmbeddingHub] The Embedding hub is up and running")
}

func (e *EmbeddingHub) Worker() {
	for {
		select {
		case job := <-e.Jobs:
			err := e.ProocessJob()
			if err != nil {
				job.Attempts++
				if job.Attempts <= job.MaxAttempts {
					waitTime := time.Duration(job.Attempts*3) * time.Second
					log.Printf("[EmbeddingJob] Failed, err -> %v retrying in %v", err, waitTime)

					go func(j *EmbeddingJob, delay time.Duration) {
						time.Sleep(delay)
						e.Jobs <- j
					}(job, waitTime)
				}
			}
		case <-e.Quit:
			return
		}
	}
}

func (e *EmbeddingHub) ProocessJob() error {
	return nil
}
