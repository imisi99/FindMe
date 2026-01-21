package core

import (
	"context"
	"log"
	"time"

	"findme/emb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Embedding interface {
	QueueUserCreate(id, bio string, skills, interests []string)
	QueueUserUpdate(id, bio string, skills, interest []string)
	QueueUserUpdateStatus(id string, status bool)
	QueueUserDelete(id string)
	QueueProjectCreate(id, title, description, uid string, skills []string)
	QueueProjectUpdate(id, title, description string, skills []string)
	QueueProjectUpdateStatus(id string, status bool)
	QueueProjectDelete(id string)
}

type EmbeddingJobType int

const (
	CreateUserEmbedding EmbeddingJobType = iota
	UpdateUserEmbedding
	UpdateUserStatus
	DeleteUserEmbedding
	CreateProjectEmbedding
	UpdateProjectEmbedding
	UpdateProjectStatus
	DeleteProjectEmbedding
)

type UserEmbedding struct {
	ID        string
	Bio       string
	Status    bool
	Skills    []string
	Interests []string
}

type ProjectEmbedding struct {
	ID          string
	Title       string
	Description string
	Status      bool
	Skills      []string
}

type EmbeddingJob struct {
	Type        EmbeddingJobType
	Attempts    int
	MaxAttempts int

	// User fields
	User *UserEmbedding

	// Project fields
	Project *ProjectEmbedding
}

type EmbeddingHub struct {
	Jobs       chan *EmbeddingJob
	Quit       chan bool
	WorkerPool int
	GRPCADDR   string
}

func NewEmbeddingHub(queueSize, workers int, addr string) *EmbeddingHub {
	return &EmbeddingHub{
		Jobs:       make(chan *EmbeddingJob, queueSize),
		Quit:       make(chan bool),
		WorkerPool: workers,
		GRPCADDR:   addr,
	}
}

func (e *EmbeddingHub) Run() {
	for range e.WorkerPool {
		go e.Worker()
	}
	log.Println("[EmbeddingHub] The Embedding hub is up and running")
}

func (e *EmbeddingHub) Worker() {
	conn, err := grpc.NewClient(e.GRPCADDR, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[EmbeddingHub] Failed to connect to ML service -> %v", err.Error())
		return
	}

	defer conn.Close()

	userClient := emb.NewUserEmbeddingServiceClient(conn)
	projectClient := emb.NewProjectEmbeddingServiceClient(conn)

	for {
		select {
		case job := <-e.Jobs:
			err := e.ProcessJob(job, userClient, projectClient)
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

func (e *EmbeddingHub) ProcessJob(job *EmbeddingJob, userClient emb.UserEmbeddingServiceClient, projectClient emb.ProjectEmbeddingServiceClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error

	switch job.Type {
	case CreateUserEmbedding:
		_, err = userClient.CreateUserEmbedding(ctx, &emb.UserEmbeddingRequest{
			UserId:    job.User.ID,
			Bio:       job.User.Bio,
			Skills:    job.User.Skills,
			Interests: job.User.Interests,
		})
	case UpdateUserEmbedding:
		_, err = userClient.UpdateUserEmbedding(ctx, &emb.UserEmbeddingRequest{
			UserId:    job.User.ID,
			Bio:       job.User.Bio,
			Skills:    job.User.Skills,
			Interests: job.User.Interests,
		})
	case UpdateUserStatus:
		_, err = userClient.UpdateUserStatus(ctx, &emb.UpdateStatusRequest{
			Id:     job.User.ID,
			Status: job.User.Status,
		})
	case DeleteUserEmbedding:
		_, err = userClient.DeleteUserEmbedding(ctx, &emb.DeleteEmbeddingRequest{
			Id: job.User.ID,
		})
	case CreateProjectEmbedding:
		_, err = projectClient.CreateProjectEmbedding(ctx, &emb.ProjectEmbeddingRequest{
			ProjectId:   job.Project.ID,
			Title:       job.Project.Title,
			Description: job.Project.Description,
			Skills:      job.Project.Skills,
			UserId:      job.User.ID,
		})
	case UpdateProjectEmbedding:
		_, err = projectClient.UpdateProjectEmbedding(ctx, &emb.ProjectEmbeddingRequest{
			ProjectId:   job.Project.ID,
			Title:       job.Project.Title,
			Description: job.Project.Description,
			Skills:      job.Project.Skills,
		})
	case UpdateProjectStatus:
		_, err = projectClient.UpdateProjectStatus(ctx, &emb.UpdateStatusRequest{
			Id:     job.Project.ID,
			Status: job.Project.Status,
		})
	case DeleteProjectEmbedding:
		_, err = projectClient.DeleteProjectEmbedding(ctx, &emb.DeleteEmbeddingRequest{
			Id: job.Project.ID,
		})
	}
	return err
}

func (e *EmbeddingHub) QueueUserCreate(id, bio string, skills, interests []string) {
	e.Jobs <- &EmbeddingJob{
		Type:        CreateUserEmbedding,
		MaxAttempts: 3,
		User: &UserEmbedding{
			ID:        id,
			Bio:       bio,
			Skills:    skills,
			Interests: interests,
		},
	}
}

func (e *EmbeddingHub) QueueUserUpdate(id, bio string, skills, interest []string) {
	e.Jobs <- &EmbeddingJob{
		Type:        UpdateUserEmbedding,
		MaxAttempts: 3,
		User: &UserEmbedding{
			ID:        id,
			Bio:       bio,
			Skills:    skills,
			Interests: interest,
		},
	}
}

func (e *EmbeddingHub) QueueUserUpdateStatus(id string, status bool) {
	e.Jobs <- &EmbeddingJob{
		Type:        UpdateUserStatus,
		MaxAttempts: 2,
		User: &UserEmbedding{
			ID:     id,
			Status: status,
		},
	}
}

func (e *EmbeddingHub) QueueUserDelete(id string) {
	e.Jobs <- &EmbeddingJob{
		Type:        DeleteUserEmbedding,
		MaxAttempts: 3,
		User: &UserEmbedding{
			ID: id,
		},
	}
}

func (e *EmbeddingHub) QueueProjectCreate(id, title, description, uid string, skills []string) {
	e.Jobs <- &EmbeddingJob{
		Type:        CreateProjectEmbedding,
		MaxAttempts: 3,
		Project: &ProjectEmbedding{
			ID:          id,
			Title:       title,
			Description: description,
			Skills:      skills,
		},
		User: &UserEmbedding{
			ID: uid,
		},
	}
}

func (e *EmbeddingHub) QueueProjectUpdate(id, title, description string, skills []string) {
	e.Jobs <- &EmbeddingJob{
		Type:        UpdateProjectEmbedding,
		MaxAttempts: 3,
		Project: &ProjectEmbedding{
			ID:          id,
			Title:       title,
			Description: description,
			Skills:      skills,
		},
	}
}

func (e *EmbeddingHub) QueueProjectUpdateStatus(id string, status bool) {
	e.Jobs <- &EmbeddingJob{
		Type:        UpdateProjectStatus,
		MaxAttempts: 2,
		Project: &ProjectEmbedding{
			ID:     id,
			Status: status,
		},
	}
}

func (e *EmbeddingHub) QueueProjectDelete(id string) {
	e.Jobs <- &EmbeddingJob{
		Type:        DeleteProjectEmbedding,
		MaxAttempts: 3,
		Project: &ProjectEmbedding{
			ID: id,
		},
	}
}
