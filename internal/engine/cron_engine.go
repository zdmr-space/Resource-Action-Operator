package engine

import (
	"context"
	"sync"
	"time"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type cronKey struct {
	ResourceAction string
	ResourceUID    types.UID
	ActionIndex    int
	Event          EventType
}

type CronEngine struct {
	client   client.Client
	executor Executor

	mu      sync.Mutex
	jobs    map[cronKey]context.CancelFunc
	started bool
}

func NewCronEngine(c client.Client, exec Executor) *CronEngine {
	return &CronEngine{
		client:   c,
		executor: exec,
		jobs:     make(map[cronKey]context.CancelFunc),
	}
}

func (c *CronEngine) Start(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.jobs == nil {
		c.jobs = make(map[cronKey]context.CancelFunc)
	}

	c.started = true
}

// EnsureForMatch wird bei JEDEM Event aufgerufen,
// registriert aber Cron-Jobs nur einmal.
func (c *CronEngine) EnsureForMatch(ctx context.Context, input MatchInput) error {
	logger := log.FromContext(ctx)

	var list opsv1alpha1.ResourceActionList
	if err := c.client.List(ctx, &list); err != nil {
		return err
	}

	for _, ra := range list.Items {
		// Selector / Event Match
		if !matchesSelector(ra.Spec.Selector, input.GVK) {
			continue
		}
		if !containsEvent(ra.Spec.Events, string(input.Event)) {
			continue
		}

		for i, action := range ra.Spec.Actions {
			if action.Mode != "schedule" {
				continue
			}
			if action.Schedule == "" {
				continue
			}

			key := cronKey{
				ResourceAction: ra.Name,
				ResourceUID:    input.Obj.GetUID(),
				ActionIndex:    i,
				Event:          input.Event,
			}

			c.mu.Lock()
			if _, exists := c.jobs[key]; exists {
				c.mu.Unlock()
				continue
			}

			jobCtx, cancel := context.WithCancel(context.Background())
			c.jobs[key] = cancel
			c.mu.Unlock()

			logger.Info("Starting cron action",
				"resourceAction", ra.Name,
				"schedule", action.Schedule,
				"name", input.Obj.GetName(),
			)

			go c.runCron(jobCtx, ra, action, input)
		}
	}

	return nil
}

func (c *CronEngine) runCron(
	ctx context.Context,
	ra opsv1alpha1.ResourceAction,
	action opsv1alpha1.ActionSpec,
	input MatchInput,
) {
	logger := log.FromContext(ctx)

	dur, err := time.ParseDuration(action.Schedule)
	if err != nil {
		logger.Error(err, "invalid cron duration", "schedule", action.Schedule)
		return
	}

	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping cron action",
				"resourceAction", ra.Name,
				"name", input.Obj.GetName(),
			)
			return

		case <-ticker.C:
			// Existiert Ressource noch?
			if input.Event != EventDelete {
				exists := &opsv1alpha1.ResourceAction{}
				err := c.client.Get(context.Background(), client.ObjectKey{
					Name:      ra.Name,
					Namespace: ra.Namespace,
				}, exists)
				if err != nil {
					logger.Info("Stopping cron, ResourceAction gone",
						"resourceAction", ra.Name)
					return
				}
			}

			logger.Info("Executing cron action",
				"resourceAction", ra.Name,
				"name", input.Obj.GetName(),
			)

			_ = c.executor.Execute(context.Background(), input)
		}
	}
}
