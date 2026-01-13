package core

import (
	"context"
	"log"
	"time"

	"findme/rec"
	"findme/schema"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Recommendation interface {
	QueueUserRecommendation(ID string)
	QueueProjectRecommendation(ID string)
	GetRecommendation(ID string, jobType RecommendationJobType) (*schema.RecResponse, error)
}

type RecommendationJobType int

const (
	UserRecommendation RecommendationJobType = iota
	ProjectRecommendation
)

type RecommendationJob struct {
	Type        RecommendationJobType
	ID          string
	Attempts    int
	MaxAttempts int
}

type RecommendationHub struct {
	Jobs     chan *RecommendationJob
	Quit     chan bool
	Workers  int
	GPRCAddr string
}

func NewRecommendationHub(workers, queuesize int, addr string) *RecommendationHub {
	return &RecommendationHub{
		Jobs:     make(chan *RecommendationJob, queuesize),
		Quit:     make(chan bool),
		Workers:  workers,
		GPRCAddr: addr,
	}
}

func (r *RecommendationHub) Run() {
	for range r.Workers {
		go r.WorkerPool()
	}
	log.Println("[gRPC Recommendation] The Recommendation Hub is up and running")
}

func (r *RecommendationHub) WorkerPool() {
	conn, err := grpc.NewClient(r.GPRCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println("[gRPC Recommendation] Failed to connect to grpc with address -> ", r.GPRCAddr)
	}

	defer conn.Close()

	client := rec.NewRecommendationServiceClient(conn)

	for {
		select {
		case job := <-r.Jobs:
			_, err := r.ProcessJob(job, client)
			if err != nil {
				job.Attempts++
				if job.Attempts <= job.MaxAttempts {
					waitTime := time.Duration(job.Attempts*3) * time.Second
					go func(job *RecommendationJob, delay time.Duration) {
						time.Sleep(delay)
						r.Jobs <- job
					}(job, waitTime)
				}
			}
		case <-r.Quit:
			return
		}
	}
}

func (r *RecommendationHub) ProcessJob(job *RecommendationJob, client rec.RecommendationServiceClient) (*schema.RecResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var err error
	res := &schema.RecResponse{}
	recRes := &rec.RecommendationResponse{}

	switch job.Type {
	case UserRecommendation:
		recRes, err = client.UserRecommendation(ctx, &rec.RecommendationRequest{
			Id: job.ID,
		})
	case ProjectRecommendation:
		recRes, err = client.ProjectRecommendation(ctx, &rec.RecommendationRequest{
			Id: job.ID,
		})
	}

	res.IDs = recRes.Ids

	return res, err
}

func (r *RecommendationHub) QueueUserRecommendation(projectID string) {
	r.Jobs <- &RecommendationJob{
		Type:        UserRecommendation,
		ID:          projectID,
		MaxAttempts: 3,
	}
}

func (r *RecommendationHub) QueueProjectRecommendation(userID string) {
	r.Jobs <- &RecommendationJob{
		Type:        ProjectRecommendation,
		ID:          userID,
		MaxAttempts: 3,
	}
}

func (r *RecommendationHub) GetRecommendation(ID string, jobType RecommendationJobType) (*schema.RecResponse, error) {
	conn, err := grpc.NewClient(r.GPRCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println("[gRPC Recommendation] Failed to connect to grpc with address -> ", r.GPRCAddr)
	}

	defer conn.Close()

	client := rec.NewRecommendationServiceClient(conn)

	job := &RecommendationJob{
		Type:        jobType,
		ID:          ID,
		MaxAttempts: 1,
	}

	res := &schema.RecResponse{}

	if res, err = r.ProcessJob(job, client); err != nil {
		return nil, err
	}
	return res, nil
}
